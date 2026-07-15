package backups

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
)

type Service struct {
	env                     config.Env
	dbgen                   *dbgen.Queries
	cr                      *cron.Cron
	executionsService       *executions.Service
	scheduledBackupsEnabled bool
}

func New(
	env config.Env,
	dbgen *dbgen.Queries,
	cr *cron.Cron,
	executionsService *executions.Service,
	scheduledBackupsEnabled bool,
) *Service {
	return &Service{
		env:                     env,
		dbgen:                   dbgen,
		cr:                      cr,
		executionsService:       executionsService,
		scheduledBackupsEnabled: scheduledBackupsEnabled,
	}
}
