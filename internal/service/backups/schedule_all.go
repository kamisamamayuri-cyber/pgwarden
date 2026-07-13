package backups

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

func (s *Service) ScheduleAll() {
	activeBackups, err := s.dbgen.BackupsServiceGetScheduleAllData(
		context.Background(),
	)
	if err != nil {
		logger.Error("error getting all active backups", logger.KV{"error": err})
		return
	}

	if !s.scheduledBackupsEnabled {
		for _, backup := range activeBackups {
			if err := s.jobRemove(backup.ID); err != nil {
				logger.Error("error removing backup cron job", logger.KV{
					"backup_id": backup.ID.String(),
					"error":     err,
				})
			}
		}
		logger.Info("scheduled backups disabled (PBW_SCHEDULED_BACKUPS_ENABLED=false)")
		return
	}

	for _, backup := range activeBackups {
		if !backup.IsActive {
			err := s.jobRemove(backup.ID)
			if err != nil {
				logger.Error("error removing inactive backup", logger.KV{"error": err})
			}
		}

		if backup.IsActive {
			err := s.jobUpsert(backup.ID, backup.TimeZone, backup.CronExpression)
			if err != nil {
				logger.Error("error scheduling backup", logger.KV{"error": err})
			}
		}
	}

	logger.Info("all active backups scheduled")
}
