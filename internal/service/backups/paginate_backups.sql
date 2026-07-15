-- name: BackupsServicePaginateBackupsCount :one
SELECT COUNT(*)
FROM backups
INNER JOIN databases ON backups.database_id = databases.id
LEFT JOIN destinations ON backups.destination_id = destinations.id
WHERE
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(pgp_sym_decrypt(databases.connection_string, sqlc.arg('encryption_key')::TEXT))
    ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
);

-- name: BackupsServicePaginateBackups :many
SELECT
  backups.*,
  databases.name AS database_name,
  destinations.name AS destination_name
FROM backups
INNER JOIN databases ON backups.database_id = databases.id
LEFT JOIN destinations ON backups.destination_id = destinations.id
WHERE
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(pgp_sym_decrypt(databases.connection_string, sqlc.arg('encryption_key')::TEXT))
    ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
)
ORDER BY backups.created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
