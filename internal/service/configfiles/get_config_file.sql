-- name: ConfigFilesServiceGetConfigFile :one
SELECT name, content, updated_at
FROM config_files
WHERE name = $1;
