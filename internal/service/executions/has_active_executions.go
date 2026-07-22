package executions

import "context"

func (s *Service) HasActiveExecutions(ctx context.Context) (bool, error) {
	return s.dbgen.ExecutionsServiceHasActiveExecutions(ctx)
}
