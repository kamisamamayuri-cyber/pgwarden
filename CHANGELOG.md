# Changelog

## Unreleased

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

### TODO

- Пересмотр архитектуры restore presets: шаблонизация через `host_map` (source host → target host по environment, порт и база совпадают)
- Fix discovery incorrectly logging PostgreSQL auth failures as `port_not_postgres` info
- Allow restore presets with no targets when `execute_group` is not set (view-only access)

## v1.0.0

- Fork of PG Back Web published as PG Warden
- UI fully translated to English
- New PG Warden shield logo
- Anonymized all internal references
- Added RBAC support for restore presets
- Added discovery events
- Added config files management
- Added serializable/deferrable backup options
