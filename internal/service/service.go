package service

import (
	"context"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/config"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/cron"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auditlogs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auth"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/backups"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/configfiles"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/databases"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/destinations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/discovery"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/jobs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/users"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/versioncheck"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
)

type Service struct {
	AuthService         *auth.Service
	AuditLogsService    *auditlogs.Service
	BackupsService      *backups.Service
	ConfigFilesService  *configfiles.Service
	DatabasesService    *databases.Service
	DestinationsService *destinations.Service
	ExecutionsService   *executions.Service
	JobsService         *jobs.Service
	UsersService        *users.Service
	RestorationsService *restorations.Service
	RbacService         *rbac.Service
	DiscoveryService    *discovery.Service
	VersionCheckService *versioncheck.Service
}

func New(
	env config.Env, db dbgen.DBTX, dbgen *dbgen.Queries,
	cr *cron.Cron, ints *integration.Integration,
) (*Service, error) {
	configFilesService := configfiles.New(dbgen, env.PBW_RESTORE_PRESETS_PATH, env.PBW_DISCOVERY_CONFIG_PATH)
	if err := configFilesService.LoadAll(context.Background()); err != nil {
		return nil, err
	}

	authService := auth.New(env, dbgen)
	databasesService := databases.New(env, dbgen, ints)
	destinationsService := destinations.New(env, dbgen, ints)
	executionsService := executions.New(env, dbgen, ints)
	usersService := users.New(dbgen)
	backupsService := backups.New(
		env, dbgen, cr, executionsService, env.PBW_SCHEDULED_BACKUPS_ENABLED,
	)
	discoveryService := discovery.New(
		configFilesService.LastDiscoveryConfig(),
		dbgen,
		databasesService,
		backupsService,
		env.PBW_DISCOVERY_PG_USER,
		env.PBW_ENCRYPTION_KEY,
		cr,
		env.PBW_DISCOVERY_SCHEDULED_ENABLED,
		env.PBW_DISCOVERY_CRON_EXPRESSION,
		env.PBW_DISCOVERY_TIME_ZONE,
		int(env.PBW_DISCOVERY_EVENTS_RETENTION_DAYS),
	)
	configFilesService.SetDiscoveryService(discoveryService)
	restorationsService := restorations.New(
		env, dbgen, ints, executionsService, databasesService, destinationsService,
	)
	// Single-process mode owns all jobs, so anything left "running" after a
	// restart is provably dead. With separate worker pods that is no longer
	// true (another pod may be mid-dump) — there the heartbeat reaper decides.
	if env.PBW_ROLE == "all" {
		executionsService.MarkStaleExecutions(context.Background())
		restorationsService.MarkStaleRestorations(context.Background())
	}

	rbacService := rbac.NewLive(env.PBW_RBAC_ADMIN_GROUP)

	return &Service{
		AuthService:         authService,
		AuditLogsService:    auditlogs.New(db, int(env.PBW_AUDIT_LOG_RETENTION_DAYS)),
		BackupsService:      backupsService,
		ConfigFilesService:  configFilesService,
		DatabasesService:    databasesService,
		DestinationsService: destinationsService,
		ExecutionsService:   executionsService,
		JobsService:         jobs.New(dbgen),
		UsersService:        usersService,
		RestorationsService: restorationsService,
		RbacService:         rbacService,
		DiscoveryService:    discoveryService,
		VersionCheckService: versioncheck.New(
			"kamisamamayuri-cyber", "pgwarden", component.AppVersion,
		),
	}, nil
}
