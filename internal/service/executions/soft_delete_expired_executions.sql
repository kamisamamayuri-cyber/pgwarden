-- name: ExecutionsServiceGetExpiredExecutions :many
-- monthly_keepers: for backups with monthly_retention_enabled=true, the
-- earliest successful execution in each (year, month) gets a separate expiry
-- window of default_monthly_retention_months (PBW_MONTHLY_RETENTION_MONTHS)
-- instead of the regular retention_days window.
WITH monthly_keepers AS (
  SELECT DISTINCT ON (executions.backup_id, date_trunc('month', executions.finished_at))
    executions.id
  FROM executions
  JOIN backups ON executions.backup_id = backups.id
  WHERE
    backups.monthly_retention_enabled = true
    AND executions.status = 'success'
    AND executions.finished_at IS NOT NULL
  ORDER BY executions.backup_id, date_trunc('month', executions.finished_at), executions.finished_at ASC
)
SELECT executions.*
FROM executions
JOIN backups ON executions.backup_id = backups.id
LEFT JOIN monthly_keepers ON monthly_keepers.id = executions.id
WHERE
  executions.status != 'deleted'
  AND executions.finished_at IS NOT NULL
  AND executions.finished_at + (
    CASE
      WHEN monthly_keepers.id IS NOT NULL
        THEN sqlc.arg('default_monthly_retention_months')::SMALLINT || ' months'
      WHEN backups.retention_days > 0
        THEN backups.retention_days || ' days'
      ELSE sqlc.arg('default_retention_days')::SMALLINT || ' days'
    END
  )::INTERVAL < NOW()
ORDER BY executions.finished_at ASC
LIMIT sqlc.arg('batch_limit');
