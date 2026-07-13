package restorations

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/databases"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/destinations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
)

type Service struct {
	env                 config.Env
	dbgen               *dbgen.Queries
	ints                *integration.Integration
	executionsService   *executions.Service
	databasesService    *databases.Service
	destinationsService *destinations.Service
}

func New(
	env config.Env,
	dbgen *dbgen.Queries, ints *integration.Integration,
	executionsService *executions.Service, databasesService *databases.Service,
	destinationsService *destinations.Service,
) *Service {
	return &Service{
		env:                 env,
		dbgen:               dbgen,
		ints:                ints,
		executionsService:   executionsService,
		databasesService:    databasesService,
		destinationsService: destinationsService,
	}
}
