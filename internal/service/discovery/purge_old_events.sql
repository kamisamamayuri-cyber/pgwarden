-- name: DiscoveryServicePurgeOldEvents :execrows
DELETE FROM discovery_events
WHERE id IN (
  SELECT discovery_events.id
  FROM discovery_events
  WHERE discovery_events.created_at < NOW() - (sqlc.arg('retention_days')::TEXT || ' days')::INTERVAL
  ORDER BY discovery_events.created_at ASC
  LIMIT sqlc.arg('batch_limit')
);
