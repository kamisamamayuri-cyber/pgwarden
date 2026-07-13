package backups

import (
	"context"

	"github.com/google/uuid"
)

func (s *Service) jobUpsert(
	backupID uuid.UUID, timeZone string, cronExpression string,
) error {
	if !s.scheduledBackupsEnabled {
		return nil
	}

	return s.cr.UpsertJob(
		backupID, timeZone, cronExpression,
		func() {
			// Enqueue only: a worker claims and runs the dump. Duplicate
			// enqueues from other pods collapse on the unique index.
			_ = s.executionsService.EnqueueExecution(context.Background(), backupID)
		},
	)
}
