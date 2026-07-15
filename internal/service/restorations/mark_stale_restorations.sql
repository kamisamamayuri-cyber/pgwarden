-- name: RestorationsServiceMarkStaleRestorations :exec
UPDATE restorations
SET
  status = 'failed',
  message = 'Restoration interrupted: process restarted while running',
  finished_at = now()
WHERE status = 'running';
