-- +goose Up

-- Close stale "running" rows first so the unique index can be created.
-- Equivalent to the MarkStaleExecutions cleanup that runs on every startup.
UPDATE executions
SET
  status = 'failed',
  message = 'Backup interrupted: process restarted while running',
  finished_at = now()
WHERE status = 'running'
  AND deleted_at IS NULL;

-- Guarantees at most one running execution per backup at the database level,
-- closing the check-then-insert race in RunExecution.
CREATE UNIQUE INDEX executions_one_running_per_backup_uidx
  ON executions (backup_id)
  WHERE status = 'running' AND deleted_at IS NULL;

-- +goose Down
DROP INDEX executions_one_running_per_backup_uidx;
