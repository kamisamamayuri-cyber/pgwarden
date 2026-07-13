-- name: ConfigFilesServiceCreateConfigFileBackup :one
INSERT INTO config_file_backups (config_name, content)
VALUES ($1, $2)
RETURNING id, config_name, content, created_at;
