package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/worker"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view"
	"github.com/labstack/echo/v4"
)

const shutdownTimeout = 30 * time.Second

func main() {
	env, err := config.GetEnv()
	if err != nil {
		logger.FatalError("error getting environment variables", logger.KV{"error": err})
	}

	pathutil.SetPathPrefix(env.PBW_PATH_PREFIX)

	ctx, stop := signal.NotifyContext(
		context.Background(), syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	cr, err := cron.New()
	if err != nil {
		logger.FatalError("error initializing cron scheduler", logger.KV{"error": err})
	}
	cr.Start()
	defer func() {
		if err := cr.Shutdown(); err != nil {
			logger.Error("error shutting down cron scheduler", logger.KV{"error": err})
		}
	}()

	db := database.Connect(env)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("error closing database connection", logger.KV{"error": err})
		}
	}()
	dbgen := dbgen.New(db)

	ints := integration.New()
	servs, err := service.New(env, dbgen, cr, ints)
	if err != nil {
		logger.FatalError("error initializing services", logger.KV{"error": err})
	}

	role := env.PBW_ROLE
	logger.Info("starting pgbackweb", logger.KV{"role": role})

	// Every pod follows config edits made on other pods (presets, discovery).
	go servs.ConfigFilesService.Watch(ctx, 30*time.Second)

	// Cron (backup schedule enqueues, discovery, housekeeping) lives with the
	// workers: enqueues are deduplicated by the DB, so every worker pod may
	// run it. Web pods serve HTTP only.
	if role == "worker" || role == "all" {
		initSchedule(cr, servs)
	}

	workerDone := make(chan struct{})
	if role == "worker" || role == "all" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "pgbackweb-worker"
		}
		wrk := worker.New(
			hostname,
			env.PBW_WORKER_CONCURRENCY,
			servs.ExecutionsService,
			servs.RestorationsService,
		)
		go func() {
			defer close(workerDone)
			wrk.Run(ctx)
		}()
	} else {
		close(workerDone)
	}

	app := echo.New()
	app.HideBanner = true
	app.HidePort = true

	switch role {
	case "worker":
		// Workers expose only a liveness endpoint for k8s probes.
		app.GET("/healthz", func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
	default:
		view.MountRouter(app, servs)
	}

	address := fmt.Sprintf("%s:%s", env.PBW_LISTEN_HOST, env.PBW_LISTEN_PORT)
	go func() {
		localURL := fmt.Sprintf("http://localhost:%s%s", env.PBW_LISTEN_PORT, pathutil.GetPathPrefix())
		logger.Info("server started at "+localURL, logger.KV{
			"listenHost": env.PBW_LISTEN_HOST,
			"listenPort": env.PBW_LISTEN_PORT,
			"role":       role,
		})
		if err := app.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.FatalError("error starting server", logger.KV{"error": err})
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received", logger.KV{"role": role})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := app.Shutdown(shutdownCtx); err != nil {
		logger.Error("error shutting down http server", logger.KV{"error": err})
	}

	// Wait for in-flight dumps/restores; k8s terminationGracePeriodSeconds is
	// the hard ceiling, after which claimed jobs are recovered by the reaper.
	select {
	case <-workerDone:
	case <-time.After(shutdownTimeout):
		logger.Warn("worker jobs still running at shutdown deadline")
	}
}
