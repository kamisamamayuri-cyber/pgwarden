-- name: ExecutionsServiceHasRunningExecution :one
SELECT EXISTS (
  SELECT 1 FROM executions
  WHERE backup_id = @backup_id AND status IN ('queued', 'running')
    AND deleted_at IS NULL
) AS running;
