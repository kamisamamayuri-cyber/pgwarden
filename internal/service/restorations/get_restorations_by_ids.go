package restorations

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetRestorationsByIDs(
	ctx context.Context, ids []uuid.UUID,
) ([]dbgen.RestorationsServicePaginateRestorationsRow, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return s.dbgen.RestorationsServicePaginateRestorations(
		ctx, dbgen.RestorationsServicePaginateRestorationsParams{
			Ids:    ids,
			Limit:  int32(len(ids)),
			Offset: 0,
		},
	)
}
