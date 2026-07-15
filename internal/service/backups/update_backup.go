package backups

import (
	"context"
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
)

func (s *Service) UpdateBackup(
	ctx context.Context, params dbgen.BackupsServiceUpdateBackupParams,
) (dbgen.Backup, error) {
	if !validate.CronExpression(params.CronExpression.String) {
		return dbgen.Backup{}, fmt.Errorf("invalid cron expression")
	}

	backup, err := s.dbgen.BackupsServiceUpdateBackup(ctx, params)
	if err != nil {
		return backup, err
	}

	if !backup.IsActive {
		return backup, s.jobRemove(backup.ID)
	}

	return backup, s.jobUpsert(backup.ID, backup.TimeZone, backup.CronExpression)
}
