-- name: ExecutionsServiceHasActiveExecutions :one
SELECT EXISTS(
  SELECT 1 FROM executions WHERE status IN ('queued', 'running')
);
