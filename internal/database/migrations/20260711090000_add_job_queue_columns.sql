-- +goose Up

-- Executions: allow 'queued' status, add worker-claim bookkeeping.
ALTER TABLE executions DROP CONSTRAINT IF EXISTS executions_status_check;
ALTER TABLE executions ADD CONSTRAINT executions_status_check CHECK (
  status IN ('queued', 'running', 'success', 'failed', 'deleted')
);
ALTER TABLE executions ADD COLUMN claimed_by TEXT;
ALTER TABLE executions ADD COLUMN heartbeat_at TIMESTAMPTZ;

-- One active (queued or running) execution per backup: dedups both concurrent
-- enqueues (cron in every worker pod) and concurrent claims.
DROP INDEX IF EXISTS executions_one_running_per_backup_uidx;
CREATE UNIQUE INDEX executions_one_active_per_backup_uidx
  ON executions (backup_id)
  WHERE status IN ('queued', 'running') AND deleted_at IS NULL;

-- Restorations: allow 'queued' status, add worker-claim bookkeeping and
-- queued-job parameters. Connection string is stored encrypted (pgcrypto),
-- params holds only non-secret restore options as JSON text.
ALTER TABLE restorations DROP CONSTRAINT IF EXISTS restorations_status_check;
ALTER TABLE restorations ADD CONSTRAINT restorations_status_check CHECK (
  status IN ('queued', 'running', 'success', 'failed')
);
ALTER TABLE restorations ADD COLUMN claimed_by TEXT;
ALTER TABLE restorations ADD COLUMN heartbeat_at TIMESTAMPTZ;
ALTER TABLE restorations ADD COLUMN params TEXT;
ALTER TABLE restorations ADD COLUMN enc_conn_string BYTEA;

DROP INDEX IF EXISTS restorations_one_running_per_target_uidx;
CREATE UNIQUE INDEX restorations_one_active_per_target_uidx
  ON restorations (target_database_name)
  WHERE status IN ('queued', 'running') AND target_database_name IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS restorations_one_active_per_target_uidx;
CREATE UNIQUE INDEX restorations_one_running_per_target_uidx
  ON restorations (target_database_name)
  WHERE status = 'running' AND target_database_name IS NOT NULL;
ALTER TABLE restorations DROP COLUMN enc_conn_string;
ALTER TABLE restorations DROP COLUMN params;
ALTER TABLE restorations DROP COLUMN heartbeat_at;
ALTER TABLE restorations DROP COLUMN claimed_by;
ALTER TABLE restorations DROP CONSTRAINT restorations_status_check;
ALTER TABLE restorations ADD CONSTRAINT restorations_status_check CHECK (
  status IN ('running', 'success', 'failed')
);

DROP INDEX IF EXISTS executions_one_active_per_backup_uidx;
CREATE UNIQUE INDEX executions_one_running_per_backup_uidx
  ON executions (backup_id)
  WHERE status = 'running' AND deleted_at IS NULL;
ALTER TABLE executions DROP COLUMN heartbeat_at;
ALTER TABLE executions DROP COLUMN claimed_by;
ALTER TABLE executions DROP CONSTRAINT executions_status_check;
ALTER TABLE executions ADD CONSTRAINT executions_status_check CHECK (
  status IN ('running', 'success', 'failed', 'deleted')
);
