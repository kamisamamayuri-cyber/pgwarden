-- name: DiscoveryServiceGetDatabaseByName :one
SELECT * FROM databases
WHERE name = @name;

-- name: DiscoveryServiceGetBackupByDestDir :one
SELECT * FROM backups
WHERE dest_dir = @dest_dir;

-- name: DiscoveryServiceCreateEvent :exec
INSERT INTO discovery_events (
  run_id, level, event, host, port, database_name, message
)
VALUES (
  @run_id, @level, @event, @host, @port, @database_name, @message
);

-- name: DiscoveryServicePaginateEventsCount :one
SELECT COUNT(discovery_events.*)
FROM discovery_events
WHERE
(
  sqlc.narg('level')::TEXT IS NULL
  OR discovery_events.level = sqlc.narg('level')::TEXT
)
AND
(
  sqlc.narg('event')::TEXT IS NULL
  OR discovery_events.event = sqlc.narg('event')::TEXT
)
AND
(
  sqlc.narg('host')::TEXT IS NULL
  OR discovery_events.host ILIKE '%' || sqlc.narg('host')::TEXT || '%'
)
AND
(
  sqlc.narg('port')::INTEGER IS NULL
  OR discovery_events.port = sqlc.narg('port')::INTEGER
)
AND
(
  sqlc.narg('database_name')::TEXT IS NULL
  OR discovery_events.database_name ILIKE '%' || sqlc.narg('database_name')::TEXT || '%'
);

-- name: DiscoveryServicePaginateEvents :many
SELECT *
FROM discovery_events
WHERE
(
  sqlc.narg('level')::TEXT IS NULL
  OR discovery_events.level = sqlc.narg('level')::TEXT
)
AND
(
  sqlc.narg('event')::TEXT IS NULL
  OR discovery_events.event = sqlc.narg('event')::TEXT
)
AND
(
  sqlc.narg('host')::TEXT IS NULL
  OR discovery_events.host ILIKE '%' || sqlc.narg('host')::TEXT || '%'
)
AND
(
  sqlc.narg('port')::INTEGER IS NULL
  OR discovery_events.port = sqlc.narg('port')::INTEGER
)
AND
(
  sqlc.narg('database_name')::TEXT IS NULL
  OR discovery_events.database_name ILIKE '%' || sqlc.narg('database_name')::TEXT || '%'
)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DiscoveryServicePaginateRunsCount :one
SELECT COUNT(DISTINCT run_id)
FROM discovery_events
WHERE
(
  sqlc.narg('level')::TEXT IS NULL
  OR discovery_events.level = sqlc.narg('level')::TEXT
)
AND
(
  sqlc.narg('event')::TEXT IS NULL
  OR discovery_events.event = sqlc.narg('event')::TEXT
)
AND
(
  sqlc.narg('host')::TEXT IS NULL
  OR discovery_events.host ILIKE '%' || sqlc.narg('host')::TEXT || '%'
)
AND
(
  sqlc.narg('port')::INTEGER IS NULL
  OR discovery_events.port = sqlc.narg('port')::INTEGER
)
AND
(
  sqlc.narg('database_name')::TEXT IS NULL
  OR discovery_events.database_name ILIKE '%' || sqlc.narg('database_name')::TEXT || '%'
);

-- name: DiscoveryServicePaginateRuns :many
SELECT
  run_id,
  MIN(created_at) AS started_at,
  MAX(created_at) AS updated_at,
  COUNT(*) FILTER (WHERE event = 'scan_finished') > 0 AS finished,
  COUNT(*) FILTER (WHERE event = 'error')::INTEGER AS errors_count,
  COUNT(*) FILTER (WHERE event = 'port_found')::INTEGER AS ports_count,
  COUNT(*) FILTER (WHERE event = 'database_found')::INTEGER AS databases_count,
  COUNT(*) FILTER (WHERE event = 'database_created')::INTEGER AS databases_created_count,
  COUNT(*) FILTER (WHERE event = 'backup_created')::INTEGER AS backups_created_count,
  COUNT(*) FILTER (WHERE event = 'skipped_existing')::INTEGER AS skipped_existing_count
FROM discovery_events
WHERE run_id IN (
  SELECT DISTINCT run_id
  FROM discovery_events
  WHERE
  (
    sqlc.narg('level')::TEXT IS NULL
    OR discovery_events.level = sqlc.narg('level')::TEXT
  )
  AND
  (
    sqlc.narg('event')::TEXT IS NULL
    OR discovery_events.event = sqlc.narg('event')::TEXT
  )
  AND
  (
    sqlc.narg('host')::TEXT IS NULL
    OR discovery_events.host ILIKE '%' || sqlc.narg('host')::TEXT || '%'
  )
  AND
  (
    sqlc.narg('port')::INTEGER IS NULL
    OR discovery_events.port = sqlc.narg('port')::INTEGER
  )
  AND
  (
    sqlc.narg('database_name')::TEXT IS NULL
    OR discovery_events.database_name ILIKE '%' || sqlc.narg('database_name')::TEXT || '%'
  )
)
GROUP BY run_id
ORDER BY MAX(created_at) DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: DiscoveryServiceHasActiveRun :one
SELECT EXISTS (
  SELECT 1
  FROM (
    SELECT run_id
    FROM discovery_events
    GROUP BY run_id
    HAVING COUNT(*) FILTER (WHERE event = 'scan_finished') = 0
       AND MAX(created_at) > NOW() - INTERVAL '30 minutes'
  ) active_runs
) AS active;

-- name: DiscoveryServiceListRunEvents :many
SELECT *
FROM discovery_events
WHERE run_id = @run_id
AND (
  (
    sqlc.arg('report_only')::BOOLEAN = FALSE
    AND event IN (
      'scan_started',
      'host_scan_started',
      'scan_progress',
      'tcp_scan_finished',
      'postgres_probe_started',
      'postgres_probe_finished',
      'port_found',
      'port_not_postgres',
      'port_probe_failed',
      'cluster_list_started',
      'cluster_list_finished',
      'database_found',
      'database_register_started',
      'database_created',
      'backup_created',
      'skipped_existing',
      'error',
      'host_scan_finished',
      'scan_finished'
    )
  )
  OR
  (
    sqlc.arg('report_only')::BOOLEAN = TRUE
    AND event IN ('database_created', 'backup_created', 'skipped_existing', 'error')
  )
)
ORDER BY created_at ASC;
