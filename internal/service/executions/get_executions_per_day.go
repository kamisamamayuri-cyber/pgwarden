package executions

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

func (s *Service) GetExecutionsPerDay(
	ctx context.Context, days int32,
) ([]dbgen.ExecutionsServiceGetExecutionsPerDayRow, error) {
	return s.dbgen.ExecutionsServiceGetExecutionsPerDay(ctx, days)
}
