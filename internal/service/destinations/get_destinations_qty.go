package destinations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetDestinationsQty(
	ctx context.Context,
) (dbgen.DestinationsServiceGetDestinationsQtyRow, error) {
	return s.dbgen.DestinationsServiceGetDestinationsQty(ctx)
}
