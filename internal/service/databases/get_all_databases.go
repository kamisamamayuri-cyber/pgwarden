package databases

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetAllDatabases(
	ctx context.Context,
) ([]dbgen.DatabasesServiceGetAllDatabasesRow, error) {
	return s.dbgen.DatabasesServiceGetAllDatabases(
		ctx, s.env.PBW_ENCRYPTION_KEY,
	)
}
