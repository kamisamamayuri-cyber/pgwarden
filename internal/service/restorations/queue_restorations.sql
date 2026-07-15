-- name: RestorationsServiceEnqueueRestoration :one
INSERT INTO restorations (
  execution_id, database_id, target_database_name, status, params, enc_conn_string
)
VALUES (
  @execution_id, @database_id, @target_database_name, 'queued', @params,
  CASE WHEN @conn_string::TEXT <> ''
    THEN pgp_sym_encrypt(@conn_string::TEXT, @encryption_key::TEXT)
  END
)
RETURNING *;

-- name: RestorationsServiceClaimRestoration :one
UPDATE restorations
SET
  status = 'running',
  claimed_by = @claimed_by,
  heartbeat_at = now(),
  started_at = now()
WHERE id = (
  SELECT id FROM restorations
  WHERE status = 'queued'
  ORDER BY started_at
  LIMIT 1
  FOR UPDATE SKIP LOCKED
)
RETURNING id, execution_id, database_id, params,
  COALESCE(
    pgp_sym_decrypt(enc_conn_string, @encryption_key::TEXT), ''
  )::TEXT AS conn_string;

-- name: RestorationsServiceHeartbeatRestoration :exec
UPDATE restorations
SET heartbeat_at = now()
WHERE id = @id AND status = 'running';

-- name: RestorationsServiceReapStaleRestorations :execrows
UPDATE restorations
SET
  status = 'failed',
  message = 'Restore interrupted: worker lost (heartbeat timed out)',
  finished_at = now()
WHERE status = 'running'
  AND heartbeat_at IS NOT NULL
  AND heartbeat_at < now() - make_interval(secs => @stale_seconds::int);
