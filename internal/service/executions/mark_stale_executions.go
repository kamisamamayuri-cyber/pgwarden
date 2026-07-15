package executions

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

func (s *Service) MarkStaleExecutions(ctx context.Context) {
	if err := s.dbgen.ExecutionsServiceMarkStaleExecutions(ctx); err != nil {
		logger.Error("failed to mark stale executions", logger.KV{"error": err})
	}
}
