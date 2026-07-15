-- +goose Up

-- Close stale "running" rows first so the unique index can be created.
-- Equivalent to the MarkStaleRestorations cleanup that runs on every startup.
UPDATE restorations
SET
  status = 'failed',
  message = 'Restore interrupted: process restarted while running',
  finished_at = now()
WHERE status = 'running';

-- Guarantees at most one running restoration per target database at the
-- database level: parallel restores would race on DROP/CREATE DATABASE.
CREATE UNIQUE INDEX restorations_one_running_per_target_uidx
  ON restorations (target_database_name)
  WHERE status = 'running' AND target_database_name IS NOT NULL;

-- +goose Down
DROP INDEX restorations_one_running_per_target_uidx;
