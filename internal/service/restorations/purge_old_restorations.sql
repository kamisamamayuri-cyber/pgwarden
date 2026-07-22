-- name: RestorationsServicePurgeOldRestorations :execrows
DELETE FROM restorations
WHERE id IN (
  SELECT restorations.id
  FROM restorations
  WHERE
    restorations.status != 'running'
    AND COALESCE(restorations.finished_at, restorations.started_at) < NOW() - (sqlc.arg('retention_days')::TEXT || ' days')::INTERVAL
  ORDER BY COALESCE(restorations.finished_at, restorations.started_at) ASC
  LIMIT sqlc.arg('batch_limit')
);
