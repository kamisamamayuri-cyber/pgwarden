-- +goose Up
CREATE TABLE audit_logs (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_email   TEXT        NOT NULL,
    action       TEXT        NOT NULL,
    preset_id    TEXT        NOT NULL,
    preset_title TEXT        NOT NULL,
    execution_id UUID,
    environment  TEXT,
    source       TEXT        NOT NULL
);

CREATE INDEX audit_logs_created_at_idx ON audit_logs (created_at DESC);

-- +goose Down
DROP TABLE audit_logs;
