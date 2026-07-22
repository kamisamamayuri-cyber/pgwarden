package executions

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

const executionHousekeepBatchLimit = 200

// HousekeepExecutions soft-deletes expired backup files/records and purges
// soft-deleted rows from the metadata database.
func (s *Service) HousekeepExecutions() {
	s.SoftDeleteExpiredExecutions()
	s.PurgeDeletedExecutions()
}

func (s *Service) SoftDeleteExpiredExecutions() {
	ctx := context.Background()

	expiredExecutions, err := s.dbgen.ExecutionsServiceGetExpiredExecutions(
		ctx, dbgen.ExecutionsServiceGetExpiredExecutionsParams{
			DefaultRetentionDays:          s.env.PBW_EXECUTION_RETENTION_DAYS,
			DefaultMonthlyRetentionMonths: s.env.PBW_MONTHLY_RETENTION_MONTHS,
			BatchLimit:                    executionHousekeepBatchLimit,
		},
	)
	if err != nil {
		logger.Error(
			"error fetching expired executions",
			logger.KV{"error": err},
		)
		return
	}

	if len(expiredExecutions) == 0 {
		return
	}

	deleted := 0
	for _, execution := range expiredExecutions {
		if err := s.SoftDeleteExecution(ctx, execution.ID); err != nil {
			logger.Error(
				"error soft deleting expired execution",
				logger.KV{"id": execution.ID.String(), "error": err},
			)
			continue
		}
		deleted++
	}

	logger.Info("expired executions soft deleted", logger.KV{
		"deleted": deleted,
		"batch":   len(expiredExecutions),
	})
}

func (s *Service) PurgeDeletedExecutions() {
	ctx := context.Background()

	purged, err := s.dbgen.ExecutionsServicePurgeDeletedExecutions(
		ctx, executionHousekeepBatchLimit,
	)
	if err != nil {
		logger.Error(
			"error purging deleted executions",
			logger.KV{"error": err},
		)
		return
	}

	if purged == 0 {
		return
	}

	logger.Info("deleted execution records purged", logger.KV{
		"purged": purged,
	})
}
