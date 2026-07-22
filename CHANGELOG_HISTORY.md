# Changelog history

Full changelog across all releases. See [CHANGELOG.md](CHANGELOG.md) for the current release notes.

## v1.2.0

### Added

- **Overview redesign**: activity timeline charts, period selector (7d/30d/90d), stat cards, and live health lists for Databases and Destinations
- Retention/pruning for `restorations`, `discovery_events`, and `audit_logs`, each with its own configurable retention period
- **Restore preflight check**: verifies target host reachability and owner role existence before a restore is queued, instead of failing mid-restore after the target DB was already dropped
- **"Keep monthly backups" toggle** on backup tasks: keeps one snapshot per month for a configurable number of months, independent of the regular retention window. Discovery can enable this by default via `discovery.yaml`
- **"New Job" wizard** on the Jobs page: first choose what to do (run a restore or fix an owner), then pick preset/environment; "Fix owner" reassigns a target database's owner without running a full restore
- **Type filter** on the Jobs page (All / Backup / Restore / Fix owner), same style as the Status filter, combinable with it
- **Worker tags**: backups and restore preset targets can carry a tag (e.g. `west`); a worker only claims jobs matching `PBW_WORKER_TAGS` (comma-separated). Untagged jobs default to `default`, and a worker with `PBW_WORKER_TAGS=default` acts as the catch-all — no automatic fallback between tags. Lets a worker pod be restricted to a network segment/region it actually has route to. Discovery can set a tag per host, or for a batch of hosts at once (`host_tags`), so onboarded databases/backups pick it up automatically
- Databases can also carry a worker tag: the periodic connectivity healthcheck now only runs against databases whose tag matches the worker's `PBW_WORKER_TAGS`, instead of every worker probing every registered database regardless of reachability
- Extra confirmation banner/dialog when restoring into an environment named `prod`
- **"Retry" action** on failed executions: enqueues a fresh execution for the same backup
- Audit logging for manual "Run backup" and "Download dump" actions
- Executions and Discovery tables now show live elapsed duration and self-refresh every 5s while something is active
- Restore status and log now update live while queued/running, instead of requiring a manual refresh
- "Fix owner" now runs as a queued task with a live log of executed ALTER statements, instead of a fire-and-forget request with no visibility
- **Update-available check**: dashboard header shows a badge when a newer GitHub release exists
- **Live log for running backups**: `pg_dump --verbose` output streams into "Execution details" while a backup is running
- Executions table and "Execution details" show the File size growing live while a backup is running
- `PBW_DUMP_COMPRESSION_LEVEL`: dump zip DEFLATE level is now configurable (1-9, default 3)
- **Parallel restore** for multi-file archives: schema first, then data files on N concurrent psql workers (`PBW_RESTORE_PARALLEL_JOBS`, default 4), then indexes/constraints. Single-file archives restore as before
- **Parallel dump** (per-backup toggle): splits table dumping across N concurrent `pg_dump` processes sharing one snapshot (`PBW_DUMP_PARALLEL_JOBS`, default 4). Produces a multi-file archive with a manual restore script inside; incompatible pg_dump options are disabled while the toggle is on
- Updated README.md to match the changes above

### Changed

- Executions tab renamed to "Jobs" (nav label, page title, `/dashboard/jobs` URL, and Overview stat cards/chart)
- **Restorations tab removed** — restore and "Fix owner" tasks now show in the Jobs list alongside backups, with a "Type" badge (Backup / Restore / Fix owner) distinguishing them. They aren't backup executions and were previously split across two separate pages/tabs with inconsistent columns
- The restore wizard's entry point moved from Restorations ("New Restore") to Jobs ("New Job"), and now opens with an explicit "what do you want to do" step instead of a small side button buried in environment selection
- **Table rows replaced with cards** on Jobs, Databases, Destinations, Discovery Runs, Configs history, and now Backups too: each item is a bordered card with status/health, the "⋮" options menu, and title at the top, and labeled stats below, instead of a wide table row
- Backups card shows "Monthly"/"Parallel ×N" badges (always visible, dimmed when off) instead of table columns; the six pg_dump option flags (data-only/schema-only/clean/if-exists/create/no-comments) moved out of the list into the Edit form only

### Removed

- Ad-hoc "Restore" action from the Executions page — bypassed RBAC and duplicated the New Job wizard, which is now the only way to start a restore from the UI
- Discovery page's manual "Refresh log" button — replaced by table auto-refresh
- Dead code: unused `GetBackupsQty` service method and SQL query
- "Finished at" column from the Executions table — redundant with Duration
- Dead code: `RestorationsService.HasActiveRestorations`/`HasActiveFixOwnerJobs`/`PaginateRestorations` and their SQL queries — superseded by the unified Jobs query

### Fixed

- OIDC callback errors now render a proper error page instead of a blank 200 response; successful callback now reliably redirects on page reload
- Restore presets with no targets are now allowed even when `execute_group` is set
- New Job wizard: environment names shown as-is, button style aligned with other primary actions, and the preset/environment lists now scroll inside the modal instead of growing it
- Discovery: benign scan results ("no open ports", "no Postgres found") now logged as `warn` instead of `error`, and the log UI properly filters/displays the `warn` level
- Overview: period cards now count failed runs too, untested databases/destinations no longer show as "offline", responsive layout and chart sizing fixed, and 7d/90d buttons now work
- Confirm dialogs on multi-step forms (New Job wizard's "Fix owner" and "Launch Restore" steps) were silently falling back to the browser's native confirm dialog instead of the app's styled one
- Jobs list: Fix owner rows had a standalone eye icon instead of the same "⋮" options menu used by Backup rows — now consistent
- Job/list cards: the Destination stat (cloud/local icon + name) was wrapping onto its own line instead of staying inline with the other stats — conflicting `inline`+`flex` classes on the same element
- List cards: stats with a copy button (Databases/Destinations connection fields) were vertically misaligned with plain-text stats on the same row
- Backups card showed "Parallel ×0" when a backup uses the default worker count (`parallel_dump_jobs=0` means "inherit `PBW_DUMP_PARALLEL_JOBS`") instead of the actual number that will run
- Execution details / Discovery log / Discovery report modals no longer close themselves a few seconds after opening while a backup or scan is running
- Execution details: the "Log (last lines)" scrollback no longer jumps to the top every 3 seconds as new lines arrive — its scroll position is now preserved across the live poll
- Executions and Discovery tables: scrolling down no longer snaps back to the top every 5 seconds while something is running
- "Fix owner" button no longer shown to view-only users
- S3 uploads no longer hang forever on a silent connection stall — a rolling no-progress timeout now fails them instead of blocking the backup indefinitely
- Backup speed: dump compression switched to a much faster implementation — the old single-threaded compressor was capping the whole dump pipeline. Archives stay standard .zip, fully compatible with existing restores
- Backup speed: S3 multipart upload tuned for the slow WAN link to the external S3 endpoint, so more parallel streams can compensate for per-connection throttling
- Owner reassignment after restore no longer fails with an out-of-memory error on databases with thousands of tables — each `ALTER ... OWNER` statement now runs in its own transaction instead of all of them in one
- Restore no longer fails when the dump references roles missing on the target — ownership/privilege statements are stripped during restore, since ownership is assigned by the restore flow itself
- "Log (last lines)" in Execution/Restoration details is now a proper log box (monospace, dark background, scrollable) instead of bare text
- Parallel dump: data from non-public schemas was being dumped twice due to a pg_dump exclude pattern bug. Fixed and verified 1:1 against classic pg_dump. **Parallel archives created before this fix are invalid** (restore would hit duplicate keys) and should be re-created

## v1.1.0

### Added

- **New Restore wizard**: "New Restore" button on the Restorations page opens a 3-step modal wizard — select preset → select environment → select backup (latest auto-selected) → Launch Restore. Respects RBAC (view/execute groups). On success shows a confirmation screen with toast notification.
- **Logs tab** (admin-only): audit log of dump downloads and restore launches with user, preset, environment, and source (UI/API)
- `can_execute` field in `GET /api/v1/restores` response per preset
- Host filter field in Executions tab: partial match on backup name
- Host filter in Databases and Backups tabs: partial match via ILIKE instead of exact position match

### Changed

- Discovery TCP port scan workers reduced from 512 to 128
- `/dashboard/restorations` restricted by RBAC: requires `view_group` or `execute_group` in at least one preset when RBAC is enabled
- Restore API (`/restores/*`) enforces RBAC per-handler: all endpoints require `view_group`; `POST /restores/:id/restore` additionally requires `execute_group`
- Sidebar navigation order updated

### Fixed

- `connect_timeout=30` added to psql connectivity test: executions now fail fast instead of hanging indefinitely when target DB is unreachable

## v1.0.0

- Fork of PG Back Web published as PG Warden
- UI fully translated to English
- New PG Warden shield logo
- Anonymized all internal references
- Added RBAC support for restore presets
- Added discovery events
- Added config files management
- Added serializable/deferrable backup options
