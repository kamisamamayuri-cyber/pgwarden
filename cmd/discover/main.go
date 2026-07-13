package main

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
)

func main() {
	env, err := config.GetEnv()
	if err != nil {
		logger.FatalError("error getting environment variables", logger.KV{"error": err})
	}

	db := database.Connect(env)
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("error closing database connection", logger.KV{"error": err})
		}
	}()

	queries := dbgen.New(db)
	cr, err := cron.New()
	if err != nil {
		logger.FatalError("error initializing cron scheduler", logger.KV{"error": err})
	}
	defer func() {
		if err := cr.Shutdown(); err != nil {
			logger.Error("error shutting down cron scheduler", logger.KV{"error": err})
		}
	}()

	servs, err := service.New(env, queries, cr, integration.New())
	if err != nil {
		logger.FatalError("error initializing services", logger.KV{"error": err})
	}

	result, err := servs.DiscoveryService.Run(context.Background())
	if err != nil {
		logger.FatalError("discovery failed", logger.KV{"error": err})
	}

	logger.Info("discovery completed", logger.KV{
		"clusters_scanned":  result.ClustersScanned,
		"databases_seen":    result.DatabasesSeen,
		"databases_created": result.DatabasesCreated,
		"backups_created":   result.BackupsCreated,
		"skipped_existing":  result.SkippedExisting,
	})
}
