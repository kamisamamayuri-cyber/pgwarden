-- name: ExecutionsServiceGetExecutionsPerDay :many
SELECT
  gs.day::DATE AS day,
  COALESCE(SUM(CASE WHEN e.status = 'success' THEN 1 ELSE 0 END), 0)::INTEGER AS success,
  COALESCE(SUM(CASE WHEN e.status = 'failed'  THEN 1 ELSE 0 END), 0)::INTEGER AS failed
FROM generate_series(
  CURRENT_DATE - (sqlc.arg('days')::INTEGER - 1),
  CURRENT_DATE,
  '1 day'::INTERVAL
) AS gs(day)
LEFT JOIN executions e
  ON DATE(e.started_at) = gs.day::DATE
  AND e.status != 'deleted'
GROUP BY gs.day
ORDER BY gs.day ASC;
