package executions

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration"
)

type Service struct {
	env   config.Env
	dbgen *dbgen.Queries
	ints  *integration.Integration
}

func New(
	env config.Env, dbgen *dbgen.Queries, ints *integration.Integration,
) *Service {
	return &Service{
		env:   env,
		dbgen: dbgen,
		ints:  ints,
	}
}
