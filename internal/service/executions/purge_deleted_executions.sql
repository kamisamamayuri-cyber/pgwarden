-- name: ExecutionsServicePurgeDeletedExecutions :execrows
DELETE FROM executions
WHERE id IN (
  SELECT executions.id
  FROM executions
  WHERE executions.status = 'deleted'
  ORDER BY executions.deleted_at ASC NULLS FIRST
  LIMIT sqlc.arg('batch_limit')
);
