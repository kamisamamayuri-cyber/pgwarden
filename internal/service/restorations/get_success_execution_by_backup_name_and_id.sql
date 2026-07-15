-- name: RestorationsServiceGetSuccessExecutionByBackupNameAndID :one
SELECT
  executions.id,
  executions.finished_at,
  executions.path,
  executions.file_size,
  databases.pg_version AS database_pg_version
FROM executions
INNER JOIN backups ON backups.id = executions.backup_id
INNER JOIN databases ON databases.id = backups.database_id
WHERE
  databases.name = @source_database_name
  AND executions.id = @execution_id
  AND executions.status = 'success'
  AND executions.path IS NOT NULL;
