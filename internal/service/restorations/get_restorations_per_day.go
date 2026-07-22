package restorations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetRestorationsPerDay(
	ctx context.Context, days int32,
) ([]dbgen.RestorationsServiceGetRestorationsPerDayRow, error) {
	return s.dbgen.RestorationsServiceGetRestorationsPerDay(ctx, days)
}
