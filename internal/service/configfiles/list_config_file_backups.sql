-- name: ConfigFilesServiceListConfigFileBackups :many
SELECT id, config_name, content, created_at
FROM config_file_backups
WHERE config_name = $1
ORDER BY created_at DESC
LIMIT 10;
