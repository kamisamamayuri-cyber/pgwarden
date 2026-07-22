package config

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
)

// validateEnv runs additional validations on the environment variables.
func validateEnv(env Env) error {
	if !validate.ListenHost(env.PBW_LISTEN_HOST) {
		return fmt.Errorf("invalid listen address %s", env.PBW_LISTEN_HOST)
	}

	if !validate.Port(env.PBW_LISTEN_PORT) {
		return fmt.Errorf("invalid listen port %s, valid values are 1-65535", env.PBW_LISTEN_PORT)
	}

	if !validate.PathPrefix(env.PBW_PATH_PREFIX) {
		return fmt.Errorf("invalid path prefix %s, must start with / and not end with / (or be empty)", env.PBW_PATH_PREFIX)
	}

	switch env.PBW_ROLE {
	case "web", "worker", "all":
	default:
		return fmt.Errorf("invalid PBW_ROLE %q, valid values: web, worker, all", env.PBW_ROLE)
	}

	if env.PBW_WORKER_CONCURRENCY < 1 || env.PBW_WORKER_CONCURRENCY > 32 {
		return fmt.Errorf("PBW_WORKER_CONCURRENCY must be between 1 and 32")
	}

	if env.PBW_DUMP_COMPRESSION_LEVEL < 1 || env.PBW_DUMP_COMPRESSION_LEVEL > 9 {
		return fmt.Errorf("PBW_DUMP_COMPRESSION_LEVEL must be between 1 and 9")
	}

	if env.PBW_DUMP_PARALLEL_JOBS < 2 || env.PBW_DUMP_PARALLEL_JOBS > 16 {
		return fmt.Errorf("PBW_DUMP_PARALLEL_JOBS must be between 2 and 16")
	}

	if env.PBW_RESTORE_PARALLEL_JOBS < 1 || env.PBW_RESTORE_PARALLEL_JOBS > 16 {
		return fmt.Errorf("PBW_RESTORE_PARALLEL_JOBS must be between 1 and 16")
	}

	if env.PBW_OIDC_ENABLED {
		if env.PBW_PUBLIC_URL == "" {
			return fmt.Errorf("PBW_PUBLIC_URL is required when PBW_OIDC_ENABLED is true")
		}
		if env.PBW_OIDC_ISSUER == "" {
			return fmt.Errorf("PBW_OIDC_ISSUER is required when PBW_OIDC_ENABLED is true")
		}
		if env.PBW_OIDC_CLIENT_ID == "" {
			return fmt.Errorf("PBW_OIDC_CLIENT_ID is required when PBW_OIDC_ENABLED is true")
		}
		if env.PBW_OIDC_CLIENT_SECRET == "" {
			return fmt.Errorf("PBW_OIDC_CLIENT_SECRET is required when PBW_OIDC_ENABLED is true")
		}
	}

	return nil
}
