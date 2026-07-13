-- name: ConfigFilesServiceUpdateConfigFile :one
UPDATE config_files
SET content = $2, updated_at = now()
WHERE name = $1
RETURNING name, content, updated_at;
