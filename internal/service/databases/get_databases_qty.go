package databases

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetDatabasesQty(
	ctx context.Context,
) (dbgen.DatabasesServiceGetDatabasesQtyRow, error) {
	return s.dbgen.DatabasesServiceGetDatabasesQty(ctx)
}
