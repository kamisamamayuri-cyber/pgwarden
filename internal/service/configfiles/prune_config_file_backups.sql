-- name: ConfigFilesServiceDeleteConfigFileBackup :exec
DELETE FROM config_file_backups WHERE id = $1;
