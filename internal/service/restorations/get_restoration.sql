-- name: RestorationsServiceGetRestoration :one
SELECT
  restorations.*,
  databases.name AS database_name,
  backups.name AS backup_name
FROM restorations
INNER JOIN executions ON executions.id = restorations.execution_id
INNER JOIN backups ON backups.id = executions.backup_id
LEFT JOIN databases ON databases.id = restorations.database_id
WHERE restorations.id = @id;
