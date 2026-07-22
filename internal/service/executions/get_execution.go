package executions

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetExecution(
	ctx context.Context, id uuid.UUID,
) (dbgen.ExecutionsServiceGetExecutionRow, error) {
	return s.dbgen.ExecutionsServiceGetExecution(ctx, id)
}
