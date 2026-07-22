-- +goose Up
ALTER TABLE executions ADD COLUMN log_tail TEXT;

-- +goose Down
ALTER TABLE executions DROP COLUMN log_tail;
