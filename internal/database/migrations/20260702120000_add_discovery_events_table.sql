-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS discovery_events (
  id UUID NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
  run_id UUID NOT NULL,
  level TEXT NOT NULL CHECK (level IN ('info', 'error')),
  event TEXT NOT NULL,
  host TEXT NOT NULL,
  port INTEGER,
  database_name TEXT,
  message TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_discovery_events_created_at
ON discovery_events(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_discovery_events_run_id
ON discovery_events(run_id);

CREATE INDEX IF NOT EXISTS idx_discovery_events_filter
ON discovery_events(level, event, host, port, database_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS discovery_events;
-- +goose StatementEnd
