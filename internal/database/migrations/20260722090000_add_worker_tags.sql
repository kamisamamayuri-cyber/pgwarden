-- +goose Up
ALTER TABLE backups ADD COLUMN tag TEXT NOT NULL DEFAULT 'default';
ALTER TABLE restorations ADD COLUMN tag TEXT NOT NULL DEFAULT 'default';

-- +goose Down
ALTER TABLE restorations DROP COLUMN tag;
ALTER TABLE backups DROP COLUMN tag;
