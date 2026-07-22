package destinations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetDestinationsHealth(
	ctx context.Context,
) ([]dbgen.DestinationsServiceGetDestinationsHealthRow, error) {
	return s.dbgen.DestinationsServiceGetDestinationsHealth(ctx)
}
