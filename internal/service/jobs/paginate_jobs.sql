-- name: JobsServicePaginateCount :one
SELECT COUNT(*) FROM (
  SELECT executions.id
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
  AND
  (
    sqlc.narg('kind')::TEXT IS NULL
    OR
    sqlc.narg('kind')::TEXT = 'backup'
  )

  UNION ALL

  SELECT restorations.id
  FROM restorations
  INNER JOIN executions e2 ON e2.id = restorations.execution_id
  INNER JOIN backups b2 ON b2.id = e2.backup_id
  WHERE sqlc.narg('backup_id')::UUID IS NULL
  AND sqlc.narg('database_id')::UUID IS NULL
  AND sqlc.narg('destination_id')::UUID IS NULL
  AND sqlc.narg('host')::TEXT IS NULL
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
    b2.name = ANY(sqlc.narg('names')::TEXT[])
  )
  AND
  (
    sqlc.narg('kind')::TEXT IS NULL
    OR
    sqlc.narg('kind')::TEXT = CASE WHEN restorations.params::JSONB->>'op' = 'fix_owner' THEN 'fix_owner' ELSE 'restore' END
  )
) combined;

-- name: JobsServicePaginateIDs :many
SELECT id, kind FROM (
  SELECT executions.id, 'backup'::TEXT AS kind, executions.started_at AS started_at
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
  AND
  (
    sqlc.narg('kind')::TEXT IS NULL
    OR
    sqlc.narg('kind')::TEXT = 'backup'
  )

  UNION ALL

  SELECT restorations.id,
    CASE WHEN restorations.params::JSONB->>'op' = 'fix_owner' THEN 'fix_owner' ELSE 'restore' END::TEXT AS kind,
    restorations.started_at AS started_at
  FROM restorations
  INNER JOIN executions e2 ON e2.id = restorations.execution_id
  INNER JOIN backups b2 ON b2.id = e2.backup_id
  WHERE sqlc.narg('backup_id')::UUID IS NULL
  AND sqlc.narg('database_id')::UUID IS NULL
  AND sqlc.narg('destination_id')::UUID IS NULL
  AND sqlc.narg('host')::TEXT IS NULL
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
    b2.name = ANY(sqlc.narg('names')::TEXT[])
  )
  AND
  (
    sqlc.narg('kind')::TEXT IS NULL
    OR
    sqlc.narg('kind')::TEXT = CASE WHEN restorations.params::JSONB->>'op' = 'fix_owner' THEN 'fix_owner' ELSE 'restore' END
  )
) combined
ORDER BY started_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: JobsServiceHasActive :one
SELECT EXISTS (
  SELECT 1 FROM executions WHERE status IN ('queued', 'running')
  UNION ALL
  SELECT 1 FROM restorations WHERE status IN ('queued', 'running')
);
