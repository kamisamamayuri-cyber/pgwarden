-- +goose Up
CREATE TABLE config_files (
    name       TEXT PRIMARY KEY,
    content    TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE config_file_backups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_name TEXT NOT NULL REFERENCES config_files(name) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX config_file_backups_config_name_created_at_idx
    ON config_file_backups (config_name, created_at DESC);

INSERT INTO config_files (name) VALUES
    ('restore-presets'),
    ('discovery');

-- +goose Down
DROP TABLE config_file_backups;
DROP TABLE config_files;
