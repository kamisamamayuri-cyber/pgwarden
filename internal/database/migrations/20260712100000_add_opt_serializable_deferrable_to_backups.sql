-- +goose Up
ALTER TABLE backups ADD COLUMN opt_serializable_deferrable BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE backups DROP COLUMN opt_serializable_deferrable;
