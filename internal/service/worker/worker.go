package worker

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
)

const (
	pollInterval      = 3 * time.Second
	heartbeatInterval = 30 * time.Second
	// reapStaleAfter must be well above heartbeatInterval so a paused worker
	// (GC, CPU throttling) is not mistaken for a dead one.
	reapStaleAfter = 5 * time.Minute
	reapInterval   = time.Minute
)

// Worker claims queued executions and restorations from the metadata DB and
// runs them. Multiple workers (pods) share the queue via FOR UPDATE SKIP
// LOCKED: each claim takes exactly one job, so load balances naturally —
// a worker busy with a long dump simply does not poll for more work.
type Worker struct {
	name              string
	concurrency       int
	tags              []string
	executionsService *executions.Service
	restorationsSvc   *restorations.Service
}

func New(
	name string,
	concurrency int,
	tags []string,
	executionsService *executions.Service,
	restorationsSvc *restorations.Service,
) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}
	if len(tags) == 0 {
		tags = []string{"default"}
	}
	return &Worker{
		name:              name,
		concurrency:       concurrency,
		tags:              tags,
		executionsService: executionsService,
		restorationsSvc:   restorationsSvc,
	}
}

// Run blocks until ctx is cancelled, then waits for in-flight jobs to finish.
func (w *Worker) Run(ctx context.Context) {
	logger.Info("worker started", logger.KV{
		"worker":      w.name,
		"concurrency": w.concurrency,
	})

	var wg sync.WaitGroup
	for range w.concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.claimLoop(ctx)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		w.reapLoop(ctx)
	}()

	wg.Wait()
	logger.Info("worker stopped", logger.KV{"worker": w.name})
}

func (w *Worker) claimLoop(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		claimed := w.claimAndRunExecution(ctx)
		if !claimed {
			claimed = w.claimAndRunRestoration(ctx)
		}
		if claimed {
			continue
		}

		// Empty queue: sleep with jitter so worker slots do not poll in
		// lock-step.
		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval + time.Duration(rand.Int63n(int64(time.Second)))):
		}
	}
}

func (w *Worker) claimAndRunExecution(ctx context.Context) bool {
	claim, ok, err := w.executionsService.ClaimExecution(ctx, w.name, w.tags)
	if err != nil {
		if ctx.Err() == nil {
			logger.Error("claim execution failed", logger.KV{"error": err.Error()})
		}
		return false
	}
	if !ok {
		return false
	}

	// The job itself must survive pod shutdown initiation: it is already
	// claimed, so let it finish within the k8s termination grace period.
	jobCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	stopHeartbeat := w.startHeartbeat(jobCtx, claim.ID, w.executionsService.HeartbeatExecution)
	defer stopHeartbeat()

	logger.Info("worker claimed execution", logger.KV{
		"worker":       w.name,
		"execution_id": claim.ID.String(),
		"backup_id":    claim.BackupID.String(),
	})
	_ = w.executionsService.RunClaimedExecution(jobCtx, claim.ID, claim.BackupID)
	return true
}

func (w *Worker) claimAndRunRestoration(ctx context.Context) bool {
	claim, ok, err := w.restorationsSvc.ClaimRestoration(ctx, w.name, w.tags)
	if err != nil {
		if ctx.Err() == nil {
			logger.Error("claim restoration failed", logger.KV{"error": err.Error()})
		}
		return false
	}
	if !ok {
		return false
	}

	jobCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()
	stopHeartbeat := w.startHeartbeat(jobCtx, claim.ID, w.restorationsSvc.HeartbeatRestoration)
	defer stopHeartbeat()

	logger.Info("worker claimed restoration", logger.KV{
		"worker":         w.name,
		"restoration_id": claim.ID.String(),
	})
	_ = w.restorationsSvc.RunClaimedRestoration(jobCtx, claim)
	return true
}

func (w *Worker) startHeartbeat(
	ctx context.Context,
	id uuid.UUID,
	beat func(context.Context, uuid.UUID) error,
) (stop func()) {
	hbCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if err := beat(hbCtx, id); err != nil && hbCtx.Err() == nil {
					logger.Error("heartbeat failed", logger.KV{
						"job_id": id.String(),
						"error":  err.Error(),
					})
				}
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// reapLoop periodically fails jobs whose worker stopped heartbeating.
// The UPDATE is idempotent, so running it in every worker pod is safe.
func (w *Worker) reapLoop(ctx context.Context) {
	ticker := time.NewTicker(reapInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := w.executionsService.ReapStaleExecutions(ctx, reapStaleAfter); err == nil && n > 0 {
				logger.Warn("reaped stale executions", logger.KV{"count": n})
			}
			if n, err := w.restorationsSvc.ReapStaleRestorations(ctx, reapStaleAfter); err == nil && n > 0 {
				logger.Warn("reaped stale restorations", logger.KV{"count": n})
			}
		}
	}
}
