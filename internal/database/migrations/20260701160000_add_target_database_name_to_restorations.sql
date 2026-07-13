-- +goose Up
-- +goose StatementBegin
ALTER TABLE restorations
  ADD COLUMN IF NOT EXISTS target_database_name TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE restorations
  DROP COLUMN IF EXISTS target_database_name;
-- +goose StatementEnd
