package restorations

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

func (s *Service) MarkStaleRestorations(ctx context.Context) {
	if err := s.dbgen.RestorationsServiceMarkStaleRestorations(ctx); err != nil {
		logger.Error("failed to mark stale restorations", logger.KV{"error": err})
	}
}
