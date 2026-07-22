-- +goose Up
-- +goose StatementBegin
ALTER TABLE backups ADD COLUMN monthly_retention_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN backups.monthly_retention_enabled IS
  'When true, the first successful execution of each calendar month is kept for PBW_MONTHLY_RETENTION_MONTHS months, independent of retention_days.';
-- +goose StatementEnd

-- +goose Down
ALTER TABLE backups DROP COLUMN monthly_retention_enabled;
