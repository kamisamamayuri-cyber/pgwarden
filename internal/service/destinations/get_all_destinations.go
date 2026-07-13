package destinations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetAllDestinations(
	ctx context.Context,
) ([]dbgen.DestinationsServiceGetAllDestinationsRow, error) {
	return s.dbgen.DestinationsServiceGetAllDestinations(
		ctx, s.env.PBW_ENCRYPTION_KEY,
	)
}
