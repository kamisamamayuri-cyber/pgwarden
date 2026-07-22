package discovery

import (
	"context"
	"strconv"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

const discoveryEventsPurgeBatchLimit = 500

// HousekeepEvents purges discovery_events rows older than
// PBW_DISCOVERY_EVENTS_RETENTION_DAYS.
func (s *Service) HousekeepEvents() {
	ctx := context.Background()

	purged, err := s.dbgen.DiscoveryServicePurgeOldEvents(
		ctx, dbgen.DiscoveryServicePurgeOldEventsParams{
			RetentionDays: strconv.Itoa(s.eventsRetentionDays),
			BatchLimit:    discoveryEventsPurgeBatchLimit,
		},
	)
	if err != nil {
		logger.Error("error purging old discovery events", logger.KV{"error": err})
		return
	}

	if purged == 0 {
		return
	}

	logger.Info("old discovery events purged", logger.KV{"purged": purged})
}
