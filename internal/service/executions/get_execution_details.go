package executions

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetExecutionDetails(
	ctx context.Context, id uuid.UUID,
) (dbgen.ExecutionsServiceGetExecutionDetailsRow, error) {
	return s.dbgen.ExecutionsServiceGetExecutionDetails(ctx, id)
}
