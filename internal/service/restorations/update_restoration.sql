-- name: RestorationsServiceUpdateRestoration :one
UPDATE restorations
SET
  status = COALESCE(sqlc.narg('status'), status),
  message = COALESCE(sqlc.narg('message'), message),
  log_tail = COALESCE(sqlc.narg('log_tail'), log_tail),
  finished_at = COALESCE(sqlc.narg('finished_at'), finished_at)
WHERE id = @id
RETURNING *;
