-- +goose Up
-- +goose StatementBegin
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS groups TEXT[] NOT NULL DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS full_access BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sessions
  DROP COLUMN IF EXISTS groups,
  DROP COLUMN IF EXISTS full_access;
-- +goose StatementEnd
