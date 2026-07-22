-- name: ExecutionsServiceClaimExecution :one
UPDATE executions
SET
  status = 'running',
  claimed_by = @claimed_by,
  heartbeat_at = now(),
  started_at = now()
WHERE id = (
  SELECT executions.id FROM executions
  INNER JOIN backups ON backups.id = executions.backup_id
  WHERE executions.status = 'queued' AND executions.deleted_at IS NULL
  AND backups.tag = ANY(@tags::TEXT[])
  ORDER BY executions.started_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING id, backup_id;

-- name: ExecutionsServiceHeartbeatExecution :exec
UPDATE executions
SET heartbeat_at = now()
WHERE id = @id AND status = 'running';

-- name: ExecutionsServiceReapStaleExecutions :execrows
UPDATE executions
SET
  status = 'failed',
  message = 'Backup interrupted: worker lost (heartbeat timed out)',
  finished_at = now()
WHERE status = 'running'
  AND deleted_at IS NULL
  AND heartbeat_at IS NOT NULL
  AND heartbeat_at < now() - make_interval(secs => @stale_seconds::int);
