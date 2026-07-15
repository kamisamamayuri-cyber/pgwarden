package config

import (
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Env struct {
	PBW_ENCRYPTION_KEY       string `env:"PBW_ENCRYPTION_KEY,required"`
	PBW_POSTGRES_CONN_STRING string `env:"PBW_POSTGRES_CONN_STRING,required"`
	PBW_LISTEN_HOST          string `env:"PBW_LISTEN_HOST" envDefault:"0.0.0.0"`
	PBW_LISTEN_PORT          string `env:"PBW_LISTEN_PORT" envDefault:"8085"`
	PBW_PATH_PREFIX          string `env:"PBW_PATH_PREFIX" envDefault:""`
	PBW_PUBLIC_URL           string `env:"PBW_PUBLIC_URL" envDefault:""`
	PBW_OIDC_ENABLED         bool   `env:"PBW_OIDC_ENABLED" envDefault:"false"`
	PBW_OIDC_ISSUER          string `env:"PBW_OIDC_ISSUER" envDefault:""`
	PBW_OIDC_CLIENT_ID       string `env:"PBW_OIDC_CLIENT_ID" envDefault:""`
	PBW_OIDC_CLIENT_SECRET   string `env:"PBW_OIDC_CLIENT_SECRET" envDefault:""`
	// Comma-separated Keycloak client IDs allowed to call HTTP API with Bearer JWT (e.g. individual).
	PBW_API_OIDC_CLIENT_IDS string `env:"PBW_API_OIDC_CLIENT_IDS" envDefault:"individual"`
	// When false, backup tasks are not registered in the in-process cron scheduler.
	// Manual runs from the UI still work. Default true (production).
	PBW_SCHEDULED_BACKUPS_ENABLED bool `env:"PBW_SCHEDULED_BACKUPS_ENABLED" envDefault:"true"`
	// Default retention for execution records and backup files when backup.retention_days=0.
	PBW_EXECUTION_RETENTION_DAYS int16 `env:"PBW_EXECUTION_RETENTION_DAYS" envDefault:"30"`
	// pg_dump --lock-wait-timeout: fail instead of waiting indefinitely for a
	// table lock. Any PostgreSQL time value ("10min", "300s"); empty disables the flag.
	PBW_DUMP_LOCK_WAIT_TIMEOUT string `env:"PBW_DUMP_LOCK_WAIT_TIMEOUT" envDefault:"10min"`
	// Process role: "web" serves only UI/API, "worker" runs cron + the dump/
	// restore queue (plus /healthz), "all" does everything in one process.
	PBW_ROLE string `env:"PBW_ROLE" envDefault:"all"`
	// How many dumps/restores one worker process runs in parallel.
	PBW_WORKER_CONCURRENCY int `env:"PBW_WORKER_CONCURRENCY" envDefault:"2"`
	// Path to YAML file with restore API presets.
	PBW_RESTORE_PRESETS_PATH string `env:"PBW_RESTORE_PRESETS_PATH" envDefault:"configs/restore-presets.yaml"`
	// Keycloak/AD group with full PG Warden access. Overrides admin_group from restore-presets.yaml when set.
	PBW_RBAC_ADMIN_GROUP string `env:"PBW_RBAC_ADMIN_GROUP" envDefault:""`
	// Path to YAML file with hosts and PostgreSQL cluster ports for DB discovery.
	PBW_DISCOVERY_CONFIG_PATH string `env:"PBW_DISCOVERY_CONFIG_PATH" envDefault:"configs/discovery.yaml"`
	// PostgreSQL user used by discovery to list databases and later run backups.
	PBW_DISCOVERY_PG_USER string `env:"PBW_DISCOVERY_PG_USER" envDefault:"pgwbackup"`
	// When true, discovery runs automatically by PBW_DISCOVERY_CRON_EXPRESSION.
	PBW_DISCOVERY_SCHEDULED_ENABLED bool `env:"PBW_DISCOVERY_SCHEDULED_ENABLED" envDefault:"true"`
	// Weekly by default: Sunday at 04:00 in PBW_DISCOVERY_TIME_ZONE.
	PBW_DISCOVERY_CRON_EXPRESSION string `env:"PBW_DISCOVERY_CRON_EXPRESSION" envDefault:"0 4 * * 0"`
	PBW_DISCOVERY_TIME_ZONE       string `env:"PBW_DISCOVERY_TIME_ZONE" envDefault:"Asia/Yekaterinburg"`
}

var (
	getEnvRes  Env
	getEnvErr  error
	getEnvOnce sync.Once
)

// GetEnv returns the environment variables.
//
// If there is an error, it will log it and exit the program.
func GetEnv(disableLogs ...bool) (Env, error) {
	getEnvOnce.Do(func() {
		// .env.role carries per-pod role vars (PBW_ROLE, PBW_WORKER_CONCURRENCY)
		// when both files are mounted as configs in k8s; .env is the shared
		// config. Real environment variables always take precedence: godotenv
		// never overrides existing env.
		_ = godotenv.Load(".env.role")
		_ = godotenv.Load()

		parsedEnv, err := env.ParseAs[Env]()
		if err != nil {
			getEnvErr = err
			return
		}

		if err := validateEnv(parsedEnv); err != nil {
			getEnvErr = err
			return
		}

		getEnvRes = parsedEnv
	})

	return getEnvRes, getEnvErr
}
