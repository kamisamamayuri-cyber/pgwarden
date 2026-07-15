-- name: ExecutionsServicePaginateExecutionsCount :one
SELECT COUNT(executions.*)
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE
(
  sqlc.narg('backup_id')::UUID IS NULL
  OR
  backups.id = sqlc.narg('backup_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  databases.id = sqlc.narg('database_id')::UUID
)
AND
(
  sqlc.narg('destination_id')::UUID IS NULL
  OR
  destinations.id = sqlc.narg('destination_id')::UUID
)
AND
(
  sqlc.narg('status')::TEXT IS NULL
  OR
  executions.status = sqlc.narg('status')::TEXT
)
AND
(
  sqlc.narg('status')::TEXT IS NOT NULL
  OR
  executions.status != 'deleted'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
)
AND
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(backups.name) ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
);

-- name: ExecutionsServicePaginateExecutions :many
SELECT
  executions.*,
  backups.name AS backup_name,
  databases.name AS database_name,
  databases.pg_version AS database_pg_version,
  destinations.name AS destination_name,
  backups.is_local AS backup_is_local
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE
(
  sqlc.narg('backup_id')::UUID IS NULL
  OR
  backups.id = sqlc.narg('backup_id')::UUID
)
AND
(
  sqlc.narg('database_id')::UUID IS NULL
  OR
  databases.id = sqlc.narg('database_id')::UUID
)
AND
(
  sqlc.narg('destination_id')::UUID IS NULL
  OR
  destinations.id = sqlc.narg('destination_id')::UUID
)
AND
(
  sqlc.narg('status')::TEXT IS NULL
  OR
  executions.status = sqlc.narg('status')::TEXT
)
AND
(
  sqlc.narg('status')::TEXT IS NOT NULL
  OR
  executions.status != 'deleted'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
)
AND
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(backups.name) ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
)
ORDER BY executions.started_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
