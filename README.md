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

| Variable                              | Required | Description                                                       |
| ------------------------------------- | :------: | ----------------------------------------------------------------- |
| `PBW_ENCRYPTION_KEY`                  |    ✓     | Encryption key (AES). Generate: `openssl rand -hex 32`            |
| `PBW_POSTGRES_CONN_STRING`            |    ✓     | PostgreSQL connection for pgwarden metadata                       |
| `PBW_ROLE`                            |          | `all` / `web` / `worker` (default: `all`)                         |
| `PBW_LISTEN_HOST`                     |          | Listen host (default: `0.0.0.0`)                                  |
| `PBW_LISTEN_PORT`                     |          | Port (default: `8085`)                                            |
| `PBW_PATH_PREFIX`                     |          | Path prefix, e.g. `/pgwarden`                                     |
| `TZ`                                  |          | Server timezone                                                   |
| `PBW_WORKER_CONCURRENCY`              |          | Parallel dumps per worker (default: `2`)                          |
| `PBW_EXECUTION_RETENTION_DAYS`        |          | Keep execution records for N days (default: `30`)                 |
| `PBW_MONTHLY_RETENTION_MONTHS`        |          | Keep one monthly snapshot per backup for N months (default: `12`) |
| `PBW_RESTORATION_RETENTION_DAYS`      |          | Keep finished restoration records for N days (default: `30`)      |
| `PBW_DISCOVERY_EVENTS_RETENTION_DAYS` |          | Keep discovery run events for N days (default: `14`)              |
| `PBW_AUDIT_LOG_RETENTION_DAYS`        |          | Keep audit log entries for N days (default: `180`)                |
| `PBW_SCHEDULED_BACKUPS_ENABLED`       |          | Enable cron scheduling (default: `true`)                          |
| `PBW_PUBLIC_URL`                      |          | Public URL — required when OIDC/SSO is enabled                    |
| `PBW_OIDC_ENABLED`                    |          | Enable OIDC/SSO (`true`/`false`)                                  |
| `PBW_OIDC_ISSUER`                     |          | OIDC provider issuer URL                                          |
| `PBW_OIDC_CLIENT_ID`                  |          | Client ID                                                         |
| `PBW_OIDC_CLIENT_SECRET`              |          | Client Secret                                                     |

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

The reaper only catches tasks whose worker process died (no heartbeat). It cannot catch a task whose worker is alive but stuck — e.g. a network stall with no timeout, where the heartbeat keeps ticking on its own timer regardless of whether the actual dump/upload is making progress. The shared S3 upload client guards against this specific case: every S3 connection carries a rolling 15-minute stall deadline (reset on each successful read/write), so a hung connection to storage fails with an error instead of blocking the whole `pg_dump → zip → S3` pipe forever. The deadline is rolling, not a total-transfer cap, so a slow-but-progressing multi-hour dump is never killed by it.

### Live progress log for running executions

`pg_dump` runs with `--verbose`, and its per-object progress lines (`pg_dump: dumping contents of table "..."`) are streamed live into `executions.log_tail`, flushed at most once every 2 seconds. The "Execution details" modal shows the last 20 lines and re-fetches every 3 seconds while the execution is still `running`, so you can watch which table a long backup is currently on instead of only finding out the outcome once it finishes or fails. Same mechanism (`internal/util/logtail`) restorations already used for their own live log.

### Retrying failed executions

The "Retry" action on a failed execution (Executions page, row menu) does **not** flip that row back to `queued` — it enqueues a brand-new execution for the same backup, the same way "Run now" on the Backups page does. The failed row is left untouched as a permanent record of that attempt (status, error message, timestamps). This keeps a full history of every attempt instead of overwriting it on retry, consistent with how restorations always create a new row rather than reusing one.

### Graceful shutdown

SIGTERM → worker stops accepting new tasks → waits for current tasks to finish (up to 30 s) → exits. Unfinished tasks are marked as `failed` by the reaper after 5 minutes.

### Cross-pod hot-reload of configs

`configfiles.Watch` compares `config_files.updated_at` in the DB with the last applied value every 30 seconds. A config change made in the UI is applied to all pods without a restart.

Backup schedules (added via the `web` pod UI) are picked up by workers through a `ScheduleAll` re-sync every 5 minutes.

### Restore presets

A restore preset describes one restore scenario: a `source` database to dump from, and one or more named `targets` (e.g. `stage`, `rc`) to restore into. Presets are configured as YAML — either the file at `PBW_RESTORE_PRESETS_PATH` (default `configs/restore-presets.yaml`, legacy) or, once edited once in the UI, the live copy stored in the `config_files` table takes over and is hot-reloaded to all pods within 30 seconds (see "Cross-pod hot-reload of configs" above).

```yaml
admin_group: pgwarden_admins # OIDC/AD group with full access to every preset

presets:
  - id: myapp
    title: myapp — restore prod dump into the selected environment
    rbac:
      view_group: pgwarden_myapp_r # can see the preset and list backups
      execute_group: pgwarden_myapp_rwx # can also launch a restore
    source:
      host: db-prod-01.example.internal
      port: 5432
      database: myapp
    targets:
      - environment: stage
        host: db-stage-01.example.internal
        port: 5432
        database: myapp
        owner: myapp # role the restored DB/schema is reassigned to
      - environment: rc
        host: db-rc-01.example.internal
        port: 5432
        database: myapp
        owner: myapp
```

Validation rules (enforced on save, both from the UI editor and `LoadPresetsFromBytes`):

- `id` and `title` are required and `id` must be unique across presets.
- `source.host`, `source.port` (1–65535) and `source.database` are required.
- `targets` is optional — a preset with no targets is valid (e.g. view-only access to backups, no restore destinations yet) and simply offers nothing to restore into.
- Each target's `environment` must be unique within the preset; `host`/`port`/`database` follow the same rules as `source`. `owner` is optional — when set, the restored database is reassigned to that role after restore (see "Restore preflight check" above, which verifies the role exists on the target cluster before anything is touched).
- `rbac.view_group` / `rbac.execute_group` are optional. When RBAC is disabled (`PBW_RBAC_ADMIN_GROUP` unset and no `admin_group` in the file), all presets are visible and executable by any authenticated user.

**Editing:** open **Configs → Restore presets** in the UI. The editor validates the YAML before saving (same `ValidatePresetsFromBytes` used by the API), keeps the last 10 versions as backups for rollback, and applies the new config to every pod without a restart.

#### Via the UI

Users with `view_group` (or `execute_group`) access see the preset on the **Restorations** page and can open the "New Restore" wizard: step 1 picks the preset, step 2 the target environment (only presets/environments the user is allowed to view are listed), step 3 the backup to restore (latest is auto-selected). Users with only `view_group` can browse but the "Launch Restore" action is hidden — `execute_group` is required to actually run it.

#### Via the REST API

```
GET  /api/v1/restores                       — presets visible to the caller (id, title, targets)
GET  /api/v1/restores/{id}                   — one preset's details
GET  /api/v1/restores/{id}/backups           — available backups for the preset's source
GET  /api/v1/restores/{id}/backups/{exec_id} — download link for one backup
GET  /api/v1/restores/{id}/restore           — available target environments
POST /api/v1/restores/{id}/restore           — launch a restore (form-encoded: environment, plus optional execution_id or finished_at)
GET  /api/v1/restorations/{uuid}             — status, duration, last 20 log lines
```

Authorization via Bearer JWT (`PBW_API_OIDC_CLIENT_IDS` controls which OIDC client IDs are accepted). RBAC is enforced per-handler the same way as the UI: no `view_group` → **404** (anti-IDOR, the caller cannot tell a hidden preset from a nonexistent one), no `execute_group` → **403** on `POST .../restore`.

```bash
curl -X POST "$PBW_PUBLIC_URL/api/v1/restores/myapp/restore" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "environment=stage"
```

### RBAC

OIDC groups → `rbac.Access`. No view permission → **404** (anti-IDOR, not 403). Presets and admin group are read live from the DB on every request — hot-reload without restart.

### Update-available check

PG Warden polls `https://api.github.com/repos/kamisamamayuri-cyber/pgwarden/releases/latest` every 6 hours (plus once at startup) and compares the tag against the build's own version (`component.AppVersion`). When a newer release exists, a pulsing "Update available" button appears in the dashboard header and a badge shows on the About page — both link straight to the GitHub release. Nothing is shown at all when up to date, or when the check hasn't succeeded yet (no false positives from a flaky GitHub API call).

`component.AppVersion` is a hand-maintained constant — there is no build-time version injection (e.g. via `ldflags`) yet, so it must be bumped to match the tag on every GitHub release push, or the check silently goes stale.

### Database discovery

Automatic discovery of PostgreSQL clusters from `configs/discovery.yaml`. Cron: weekly (Sun 04:00 server time). Manual run available from the UI.

Each scan writes structured events (`info` / `warn` / `error`) per host and port, viewable and filterable in the UI. A host with no open PostgreSQL ports, or a port that rejects the probe user's credentials, is logged as `warn` rather than `error` — it does not count as a scan failure.

### Restore preflight check

Before a restore is queued, PG Warden checks that the target host is reachable and, when the preset sets an `owner`, that the owner role already exists on the target cluster. Both checks run against the target's maintenance database (`postgres`) since the application database may not exist yet.

This runs **before** anything is enqueued: a missing owner role would otherwise only surface deep in the worker, after the target database has already been dropped in preparation for restore. Free disk space is not checked — PG Warden has no shell/VM access to the target host, only a PostgreSQL connection.

### Data retention

Finished `restorations` records, `discovery_events`, and `audit_logs` are purged in batches every 10 minutes, alongside the existing execution housekeeping job. Retention windows are configurable via `PBW_RESTORATION_RETENTION_DAYS` (default 30), `PBW_DISCOVERY_EVENTS_RETENTION_DAYS` (default 14), and `PBW_AUDIT_LOG_RETENTION_DAYS` (default 180 — longer, since audit logs are often needed for compliance review). Running restorations are never purged.

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

| Variable                              | Purpose                                    | Default |
| ------------------------------------- | ------------------------------------------ | ------- |
| `PBW_ROLE`                            | Pod role: `web` / `worker` / `all`         | `all`   |
| `PBW_WORKER_CONCURRENCY`              | Parallel tasks per worker                  | `2`     |
| `PBW_DUMP_LOCK_WAIT_TIMEOUT`          | `--lock-wait-timeout` for pg_dump          | `10min` |
| `PBW_DUMP_COMPRESSION_LEVEL`          | Dump zip DEFLATE level, 1-9                | `3`     |
| `PBW_DUMP_PARALLEL_JOBS`              | Default jobs for parallel dump, 2-16       | `4`     |
| `PBW_RESTORE_PARALLEL_JOBS`           | Jobs for restoring multi-file archives     | `4`     |
| `PBW_OIDC_*`                          | SSO / OIDC provider                        | —       |
| `PBW_API_OIDC_CLIENT_IDS`             | Allowed client_id values for Bearer auth   | —       |
| `PBW_RESTORE_PRESETS_PATH`            | Path to `restore-presets.yaml` (legacy)    | —       |
| `PBW_RBAC_ADMIN_GROUP`                | OIDC admin group                           | —       |
| `PBW_SCHEDULED_BACKUPS_ENABLED`       | Enable backup cron                         | `true`  |
| `PBW_DISCOVERY_*`                     | Discovery: user, cron, config path         | —       |
| `PBW_RESTORATION_RETENTION_DAYS`      | Retention for finished restoration records | `30`    |
| `PBW_DISCOVERY_EVENTS_RETENTION_DAYS` | Retention for discovery run events         | `14`    |
| `PBW_AUDIT_LOG_RETENTION_DAYS`        | Retention for audit log entries            | `180`   |

---

## License

AGPL v3 — inherited from upstream [PG Back Web](https://github.com/eduardolat/pgbackweb).
