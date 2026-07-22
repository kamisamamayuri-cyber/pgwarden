package executions

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetExecutionsByIDs(
	ctx context.Context, ids []uuid.UUID,
) ([]dbgen.ExecutionsServicePaginateExecutionsRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return s.dbgen.ExecutionsServicePaginateExecutions(
		ctx, dbgen.ExecutionsServicePaginateExecutionsParams{
			Ids:    ids,
			Limit:  int32(len(ids)),
			Offset: 0,
		},
	)
}
