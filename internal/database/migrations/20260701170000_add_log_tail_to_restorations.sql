-- +goose Up
-- +goose StatementBegin
ALTER TABLE restorations
  ADD COLUMN IF NOT EXISTS log_tail TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE restorations
  DROP COLUMN IF EXISTS log_tail;
-- +goose StatementEnd
