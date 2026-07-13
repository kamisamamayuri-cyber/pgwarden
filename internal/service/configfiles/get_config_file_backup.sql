-- name: ConfigFilesServiceGetConfigFileBackup :one
SELECT id, config_name, content, created_at
FROM config_file_backups
WHERE id = $1;
