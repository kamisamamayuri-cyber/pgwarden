-- +goose Up
ALTER TABLE backups ADD COLUMN parallel_dump_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE backups ADD COLUMN parallel_dump_jobs SMALLINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE backups DROP COLUMN parallel_dump_jobs;
ALTER TABLE backups DROP COLUMN parallel_dump_enabled;
