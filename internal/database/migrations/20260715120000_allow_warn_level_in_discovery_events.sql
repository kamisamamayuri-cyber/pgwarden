-- +goose Up
-- +goose StatementBegin
ALTER TABLE discovery_events
  DROP CONSTRAINT discovery_events_level_check;

ALTER TABLE discovery_events
  ADD CONSTRAINT discovery_events_level_check
  CHECK (level IN ('info', 'warn', 'error'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE discovery_events SET level = 'info' WHERE level = 'warn';

ALTER TABLE discovery_events
  DROP CONSTRAINT discovery_events_level_check;

ALTER TABLE discovery_events
  ADD CONSTRAINT discovery_events_level_check
  CHECK (level IN ('info', 'error'));
-- +goose StatementEnd
