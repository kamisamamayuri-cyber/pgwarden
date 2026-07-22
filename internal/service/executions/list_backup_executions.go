package executions

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) ListBackupExecutions(
	ctx context.Context, backupID uuid.UUID,
) ([]dbgen.Execution, error) {
	return s.dbgen.ExecutionsServiceListBackupExecutions(ctx, backupID)
}
