# pgwarden

PostgreSQL backup service with a web UI, scheduler, discovery, and S3 support.

- **Repository:** [github.com/kamisamamayuri-cyber/pgwarden](https://github.com/kamisamamayuri-cyber/pgwarden)

Fork of [PG Back Web](https://github.com/eduardolat/pgbackweb) (AGPL v3). Upstream is referenced only in commit history and the license.

---

## Quick start (Docker Compose)

Minimal example — a single container with PostgreSQL for metadata and local backup storage.

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16
    environment:
      POSTGRES_DB: pgwarden
      POSTGRES_USER: pgwarden
      POSTGRES_PASSWORD: pgwarden_secret
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  pgwarden:
    image: ghcr.io/kamisamamayuri-cyber/pgwarden:latest
    ports:
      - "8085:8085"
    environment:
      # Encryption key for sensitive data (connection strings, S3 keys).
      # Generate with: openssl rand -hex 32
      PBW_ENCRYPTION_KEY: "replace-with-a-random-string"

      # Connection string for the pgwarden metadata database
      PBW_POSTGRES_CONN_STRING: "postgresql://pgwarden:pgwarden_secret@postgres:5432/pgwarden?sslmode=disable"

      PBW_LISTEN_HOST: "0.0.0.0"
      PBW_LISTEN_PORT: "8085"

      # Pod role: all = web + worker in a single container
      PBW_ROLE: "all"

      TZ: "UTC"
    volumes:
      # Local backup storage (if not using S3)
      - backups:/backups
    depends_on:
      - postgres
    restart: unless-stopped

volumes:
  postgres_data:
  backups:
```

Start:

```bash
docker compose up -d
```

The web UI will be available at `http://localhost:8085`. On first launch it will prompt you to create an admin user.

Database migrations are applied automatically on container start.

### Split roles (production)

For horizontal scaling, run web and worker separately:

```yaml
pgwarden-web:
  image: ghcr.io/kamisamamayuri-cyber/pgwarden:latest
  environment:
    PBW_ROLE: "web"
    # ... other variables
  ports:
    - "8085:8085"

pgwarden-worker:
  image: ghcr.io/kamisamamayuri-cyber/pgwarden:latest
  environment:
    PBW_ROLE: "worker"
    PBW_WORKER_CONCURRENCY: "4" # parallel dumps (1–32)
    # ... other variables
```

### Environment variables

| Variable                        | Required | Description                                            |
| ------------------------------- | :------: | ------------------------------------------------------ |
| `PBW_ENCRYPTION_KEY`            |    ✓     | Encryption key (AES). Generate: `openssl rand -hex 32` |
| `PBW_POSTGRES_CONN_STRING`      |    ✓     | PostgreSQL connection for pgwarden metadata            |
| `PBW_ROLE`                      |          | `all` / `web` / `worker` (default: `all`)              |
| `PBW_LISTEN_HOST`               |          | Listen host (default: `0.0.0.0`)                       |
| `PBW_LISTEN_PORT`               |          | Port (default: `8085`)                                 |
| `PBW_PATH_PREFIX`               |          | Path prefix, e.g. `/pgwarden`                          |
| `TZ`                            |          | Server timezone                                        |
| `PBW_WORKER_CONCURRENCY`        |          | Parallel dumps per worker (default: `2`)               |
| `PBW_EXECUTION_RETENTION_DAYS`  |          | Keep execution records for N days (default: `30`)      |
| `PBW_SCHEDULED_BACKUPS_ENABLED` |          | Enable cron scheduling (default: `true`)               |
| `PBW_PUBLIC_URL`                |          | Public URL — required when OIDC/SSO is enabled         |
| `PBW_OIDC_ENABLED`              |          | Enable OIDC/SSO (`true`/`false`)                       |
| `PBW_OIDC_ISSUER`               |          | OIDC provider issuer URL                               |
| `PBW_OIDC_CLIENT_ID`            |          | Client ID                                              |
| `PBW_OIDC_CLIENT_SECRET`        |          | Client Secret                                          |

---

## Features

### Horizontal scaling: pod roles

The application supports three modes, configured via `PBW_ROLE`:

| Role     | What it does                                               |
| -------- | ---------------------------------------------------------- |
| `web`    | HTTP UI and API, no cron or dumps                          |
| `worker` | cron (enqueue) + queue processing + `/healthz`             |
| `all`    | everything in one process (default; for local development) |

Typical k8s deployment: one `frontend` pod (`PBW_ROLE=web`) and one or more `backend` pods (`PBW_ROLE=worker`). Pods communicate only through the metadata DB — no message broker needed.

### PostgreSQL-based task queue

Cron and the UI do not run dumps directly — they enqueue a task (status `queued`). Worker pods drain the queue using a work-stealing pattern:

```sql
UPDATE executions SET status='running', claimed_by=$1, heartbeat_at=now()
WHERE id = (SELECT id FROM executions WHERE status='queued' ORDER BY started_at FOR UPDATE SKIP LOCKED LIMIT 1)
```

- **Deduplication:** partial unique index on `status IN ('queued','running')` — re-enqueuing the same backup is silently ignored; cron can run on all workers without a leader.
- **Heartbeat:** worker updates `heartbeat_at` every 30 seconds.
- **Reaper:** every minute, marks tasks from pods with no heartbeat for >5 min as `failed`. Idempotent, runs on every worker.
- **`PBW_WORKER_CONCURRENCY`** — how many tasks a single worker pod processes in parallel (default 2, range 1–32). Sets the number of slots, not CPU affinity.

### Graceful shutdown

SIGTERM → worker stops accepting new tasks → waits for current tasks to finish (up to 30 s) → exits. Unfinished tasks are marked as `failed` by the reaper after 5 minutes.

### Cross-pod hot-reload of configs

`configfiles.Watch` compares `config_files.updated_at` in the DB with the last applied value every 30 seconds. A config change made in the UI is applied to all pods without a restart.

Backup schedules (added via the `web` pod UI) are picked up by workers through a `ScheduleAll` re-sync every 5 minutes.

### REST API for restore

```
POST /api/v1/restores/{id}/restore
GET  /api/v1/restorations/{uuid}   — status, duration, last 20 log lines
GET  /api/v1/restores              — list of backups available to the user
```

Authorization via Bearer JWT. Per-environment permissions are configured in `restore-presets.yaml` (stored in the DB, editable in the UI under "Configs").

### RBAC

OIDC groups → `rbac.Access`. No view permission → **404** (anti-IDOR, not 403). Presets and admin group are read live from the DB on every request — hot-reload without restart.

### Database discovery

Automatic discovery of PostgreSQL clusters from `configs/discovery.yaml`. Cron: weekly (Sun 04:00 server time). Manual run available from the UI.

### Security

- SSO via OIDC; session cookie `SameSite=Lax` + `Secure` based on request scheme
- Bearer: validates `typ=Bearer`, `nbf`, clock skew 30 s
- Rate-limit on login without holding a mutex during bcrypt
- `pg_dump` runs with `--lock-wait-timeout` (env `PBW_DUMP_LOCK_WAIT_TIMEOUT`, default `10min`)
- Cancellable processes: `exec.CommandContext` everywhere (pg_dump, psql, pg_isready)
- `sanitizeDumpReader` does not touch lines inside `COPY … FROM stdin` blocks

---

## Development

```bash
# build
task build

# generate DB code (sqlc + goose)
task gen:db

# tests
task test

# local run with migrations
task servem
```

**sqlc:** SQL queries live in `internal/service/**/*.sql`. The `scripts/sqlc-prebuild.ts` script concatenates them into `queries.gen.sql`, then runs `sqlc generate`. Files in `dbgen/` are in `.gitignore` — **a new SQL query must be in the source `.sql` file**, otherwise CI will fail.

**Migrations:** `internal/database/migrations/` (Goose), applied automatically on startup.

---

## Configuration

| Variable                        | Purpose                                  | Default |
| ------------------------------- | ---------------------------------------- | ------- |
| `PBW_ROLE`                      | Pod role: `web` / `worker` / `all`       | `all`   |
| `PBW_WORKER_CONCURRENCY`        | Parallel tasks per worker                | `2`     |
| `PBW_DUMP_LOCK_WAIT_TIMEOUT`    | `--lock-wait-timeout` for pg_dump        | `10min` |
| `PBW_OIDC_*`                    | SSO / OIDC provider                      | —       |
| `PBW_API_OIDC_CLIENT_IDS`       | Allowed client_id values for Bearer auth | —       |
| `PBW_RESTORE_PRESETS_PATH`      | Path to `restore-presets.yaml` (legacy)  | —       |
| `PBW_RBAC_ADMIN_GROUP`          | OIDC admin group                         | —       |
| `PBW_SCHEDULED_BACKUPS_ENABLED` | Enable backup cron                       | `true`  |
| `PBW_DISCOVERY_*`               | Discovery: user, cron, config path       | —       |

---

## License

AGPL v3 — inherited from upstream [PG Back Web](https://github.com/eduardolat/pgbackweb).
