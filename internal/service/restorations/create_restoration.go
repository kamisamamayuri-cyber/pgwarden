package restorations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) CreateRestoration(
	ctx context.Context, params dbgen.RestorationsServiceCreateRestorationParams,
) (dbgen.Restoration, error) {
	return s.dbgen.RestorationsServiceCreateRestoration(ctx, params)
}
