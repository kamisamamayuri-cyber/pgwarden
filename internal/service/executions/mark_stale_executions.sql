-- name: ExecutionsServiceMarkStaleExecutions :exec
UPDATE executions
SET
  status = 'failed',
  message = 'Backup interrupted: process restarted while running',
  finished_at = now()
WHERE status = 'running'
  AND deleted_at IS NULL;
