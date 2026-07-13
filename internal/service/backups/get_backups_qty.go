package backups

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetBackupsQty(
	ctx context.Context,
) (dbgen.BackupsServiceGetBackupsQtyRow, error) {
	return s.dbgen.BackupsServiceGetBackupsQty(ctx)
}
