-- name: ExecutionsServiceGetExecutionDetails :one
SELECT
  executions.*,
  backups.name AS backup_name,
  databases.name AS database_name,
  destinations.name AS destination_name,
  backups.is_local AS backup_is_local
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
LEFT JOIN destinations ON destinations.id = backups.destination_id
WHERE executions.id = @id;
