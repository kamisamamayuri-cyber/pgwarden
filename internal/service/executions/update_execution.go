package executions

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) UpdateExecution(
	ctx context.Context, params dbgen.ExecutionsServiceUpdateExecutionParams,
) (dbgen.Execution, error) {
	return s.dbgen.ExecutionsServiceUpdateExecution(ctx, params)
}
