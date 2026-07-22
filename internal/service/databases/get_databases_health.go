package databases

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetDatabasesHealth(
	ctx context.Context,
) ([]dbgen.DatabasesServiceGetDatabasesHealthRow, error) {
	return s.dbgen.DatabasesServiceGetDatabasesHealth(ctx)
}
