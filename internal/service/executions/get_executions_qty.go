package executions

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetExecutionsQty(
	ctx context.Context,
) (dbgen.ExecutionsServiceGetExecutionsQtyRow, error) {
	return s.dbgen.ExecutionsServiceGetExecutionsQty(ctx)
}
