package restorations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) UpdateRestoration(
	ctx context.Context, params dbgen.RestorationsServiceUpdateRestorationParams,
) (dbgen.Restoration, error) {
	return s.dbgen.RestorationsServiceUpdateRestoration(ctx, params)
}
