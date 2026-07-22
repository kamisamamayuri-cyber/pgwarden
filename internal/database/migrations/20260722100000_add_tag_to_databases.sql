-- +goose Up
ALTER TABLE databases ADD COLUMN tag TEXT NOT NULL DEFAULT 'default';

-- +goose Down
ALTER TABLE databases DROP COLUMN tag;
