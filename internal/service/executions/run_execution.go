package executions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration/postgres"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
)

// EnqueueExecution puts a backup execution into the queue. A worker picks it
// up via ClaimExecution. The partial unique index
// executions_one_active_per_backup_uidx guarantees at most one queued or
// running execution per backup, so concurrent enqueues (cron in every worker
// pod, manual runs) collapse into one job.
func (s *Service) EnqueueExecution(ctx context.Context, backupID uuid.UUID) error {
	running, err := s.dbgen.ExecutionsServiceHasRunningExecution(ctx, backupID)
	if err != nil {
		return err
	}
	if running {
		return fmt.Errorf("backup %s is already queued or running", backupID)
	}

	_, err = s.CreateExecution(ctx, dbgen.ExecutionsServiceCreateExecutionParams{
		BackupID: backupID,
		Status:   "queued",
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return fmt.Errorf("backup %s is already queued or running", backupID)
		}
		return err
	}
	return nil
}

// ClaimExecution atomically claims one queued execution for a worker.
// Returns ok=false when the queue is empty.
func (s *Service) ClaimExecution(
	ctx context.Context, claimedBy string,
) (dbgen.ExecutionsServiceClaimExecutionRow, bool, error) {
	row, err := s.dbgen.ExecutionsServiceClaimExecution(
		ctx, sql.NullString{Valid: true, String: claimedBy},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return row, false, nil
	}
	if err != nil {
		return row, false, err
	}
	return row, true, nil
}

// HeartbeatExecution refreshes the liveness timestamp of a running execution.
func (s *Service) HeartbeatExecution(ctx context.Context, id uuid.UUID) error {
	return s.dbgen.ExecutionsServiceHeartbeatExecution(ctx, id)
}

// ReapStaleExecutions fails running executions whose worker stopped
// heartbeating (crashed or was killed without cleanup).
func (s *Service) ReapStaleExecutions(
	ctx context.Context, staleAfter time.Duration,
) (int64, error) {
	return s.dbgen.ExecutionsServiceReapStaleExecutions(
		ctx, int32(staleAfter/time.Second),
	)
}

// RunClaimedExecution runs a backup execution that was already claimed
// (status=running). The heartbeat is maintained by the caller (worker).
func (s *Service) RunClaimedExecution(
	ctx context.Context, execID, backupID uuid.UUID,
) error {
	updateExec := func(params dbgen.ExecutionsServiceUpdateExecutionParams) {
		_, err := s.dbgen.ExecutionsServiceUpdateExecution(ctx, params)
		if err != nil {
			logger.Error("failed to update execution status", logger.KV{
				"execution_id": params.ID.String(),
				"error":        err.Error(),
			})
		}
	}

	logError := func(err error) {
		logger.Error("error running backup", logger.KV{
			"backup_id": backupID.String(),
			"error":     err.Error(),
		})
	}

	failExec := func(execID uuid.UUID, origErr error, path string) error {
		logError(origErr)
		p := dbgen.ExecutionsServiceUpdateExecutionParams{
			ID:         execID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: origErr.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		}
		if path != "" {
			p.Path = sql.NullString{Valid: true, String: path}
		}
		updateExec(p)
		return origErr
	}

	back, err := s.dbgen.ExecutionsServiceGetBackupData(
		ctx, dbgen.ExecutionsServiceGetBackupDataParams{
			BackupID:      backupID,
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		return failExec(execID, err, "")
	}

	logger.Info("backup started", logger.KV{
		"backup_id":     backupID.String(),
		"backup_name":   back.BackupName,
		"database_name": back.DatabaseName,
		"execution_id":  execID.String(),
		"dest_dir":      back.BackupDestDir,
		"is_local":      back.BackupIsLocal,
		"pg_version":    back.DatabasePgVersion,
	})

	if !back.BackupIsLocal {
		err = s.ints.StorageClient.S3Test(
			ctx,
			back.DecryptedDestinationAccessKey, back.DecryptedDestinationSecretKey,
			back.DestinationRegion.String, back.DestinationEndpoint.String,
			back.DestinationBucketName.String,
		)
		if err != nil {
			return failExec(execID, err, "")
		}
	}

	pgVersion, err := s.ints.PGClient.ParseVersion(back.DatabasePgVersion)
	if err != nil {
		return failExec(execID, err, "")
	}

	err = s.ints.PGClient.Test(ctx, pgVersion, back.DecryptedDatabaseConnectionString)
	if err != nil {
		return failExec(execID, err, "")
	}

	dumpReader := s.ints.PGClient.DumpZip(
		ctx, pgVersion, back.DecryptedDatabaseConnectionString, postgres.DumpParams{
			DataOnly:               back.BackupOptDataOnly,
			SchemaOnly:             back.BackupOptSchemaOnly,
			Clean:                  back.BackupOptClean,
			IfExists:               back.BackupOptIfExists,
			Create:                 back.BackupOptCreate,
			NoComments:             back.BackupOptNoComments,
			LockWaitTimeout:        s.env.PBW_DUMP_LOCK_WAIT_TIMEOUT,
			SerializableDeferrable: back.BackupOptSerializableDeferrable,
		},
	)

	date := time.Now().Format(timeutil.LayoutSlashYYYYMMDD)
	file := fmt.Sprintf(
		"dump-%s-%s.zip",
		time.Now().Format(timeutil.LayoutYYYYMMDDHHMMSS),
		uuid.NewString(),
	)
	path := strutil.CreatePath(false, back.BackupDestDir, date, file)
	fileSize := int64(0)

	if back.BackupIsLocal {
		fileSize, err = s.ints.StorageClient.LocalUpload(path, dumpReader)
		if err != nil {
			return failExec(execID, err, path)
		}
	}

	if !back.BackupIsLocal {
		fileSize, err = s.ints.StorageClient.S3Upload(
			ctx,
			back.DecryptedDestinationAccessKey, back.DecryptedDestinationSecretKey,
			back.DestinationRegion.String, back.DestinationEndpoint.String,
			back.DestinationBucketName.String, path, dumpReader,
		)
		if err != nil {
			return failExec(execID, err, path)
		}
	}

	logger.Info("backup finished successfully", logger.KV{
		"backup_id":     backupID.String(),
		"backup_name":   back.BackupName,
		"database_name": back.DatabaseName,
		"execution_id":  execID.String(),
		"path":          path,
		"file_size":     fileSize,
	})
	updateExec(dbgen.ExecutionsServiceUpdateExecutionParams{
		ID:         execID,
		Status:     sql.NullString{Valid: true, String: "success"},
		Message:    sql.NullString{Valid: true, String: "Backup created successfully"},
		Path:       sql.NullString{Valid: true, String: path},
		FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		FileSize:   sql.NullInt64{Valid: true, Int64: fileSize},
	})
	return nil
}
