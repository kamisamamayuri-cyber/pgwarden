-- name: RestorationsServicePaginateRestorations :many
SELECT
  restorations.*,
  databases.name AS database_name,
  backups.name AS backup_name
FROM restorations
INNER JOIN executions ON executions.id = restorations.execution_id
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN databases ON databases.id = restorations.database_id
WHERE
(
  sqlc.narg('execution_id')::UUID IS NULL
  OR
  restorations.execution_id = sqlc.narg('execution_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  restorations.database_id = sqlc.narg('database_id')::UUID
)
AND
(
  sqlc.narg('status')::TEXT IS NULL
  OR
  restorations.status = sqlc.narg('status')::TEXT
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  backups.name = ANY(sqlc.narg('names')::TEXT[])
)
AND
(
  sqlc.narg('ids')::UUID[] IS NULL
  OR
  restorations.id = ANY(sqlc.narg('ids')::UUID[])
)
ORDER BY restorations.started_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
