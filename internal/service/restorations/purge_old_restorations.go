package restorations

import (
	"context"
	"strconv"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

const restorationPurgeBatchLimit = 200

// HousekeepRestorations purges finished restoration records older than
// PBW_RESTORATION_RETENTION_DAYS. Running restorations are never touched.
func (s *Service) HousekeepRestorations() {
	ctx := context.Background()

	purged, err := s.dbgen.RestorationsServicePurgeOldRestorations(
		ctx, dbgen.RestorationsServicePurgeOldRestorationsParams{
			RetentionDays: strconv.Itoa(int(s.env.PBW_RESTORATION_RETENTION_DAYS)),
			BatchLimit:    restorationPurgeBatchLimit,
		},
	)
	if err != nil {
		logger.Error("error purging old restorations", logger.KV{"error": err})
		return
	}

	if purged == 0 {
		return
	}

	logger.Info("old restoration records purged", logger.KV{"purged": purged})
}
