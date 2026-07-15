-- name: DatabasesServicePaginateDatabasesCount :one
SELECT COUNT(*)
FROM databases
WHERE
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(pgp_sym_decrypt(connection_string, sqlc.arg('encryption_key')::TEXT))
    ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
);

-- name: DatabasesServicePaginateDatabases :many
SELECT
  *,
  pgp_sym_decrypt(connection_string, @encryption_key) AS decrypted_connection_string
FROM databases
WHERE
(
  sqlc.narg('host')::TEXT IS NULL
  OR
  lower(pgp_sym_decrypt(connection_string, sqlc.arg('encryption_key')::TEXT))
    ILIKE '%' || lower(sqlc.narg('host')::TEXT) || '%'
)
AND
(
  sqlc.narg('names')::TEXT[] IS NULL
  OR
  databases.name = ANY(sqlc.narg('names')::TEXT[])
)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
