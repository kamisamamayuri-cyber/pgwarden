package restorations

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration/postgres"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/logtail"
)

const (
	restorePhasePreparing = "preparing target database"
	restorePhaseRestoring = "restoring backup"
	restorePhaseOwner     = "reassigning owner"
)

// RunRestoration runs a backup restoration and returns the restoration record id.
func (s *Service) RunRestoration(
	ctx context.Context, params RunRestorationParams,
) (uuid.UUID, error) {
	logError := func(err error) {
		dbID := "empty"
		if params.DatabaseID.Valid {
			dbID = params.DatabaseID.UUID.String()
		}
		logger.Error("error running restoration", logger.KV{
			"execution_id": params.ExecutionID.String(),
			"database_id":  dbID,
			"error":        err.Error(),
		})
	}

	var res dbgen.Restoration
	if params.ExistingRestorationID.Valid {
		res.ID = params.ExistingRestorationID.UUID
	} else {
		targetDatabaseName := sql.NullString{}
		if params.DatabaseID.Valid {
			db, dbErr := s.databasesService.GetDatabase(ctx, params.DatabaseID.UUID)
			if dbErr == nil {
				targetDatabaseName = sql.NullString{Valid: true, String: db.Name}
			}
		}

		created, err := s.CreateRestoration(ctx, dbgen.RestorationsServiceCreateRestorationParams{
			ExecutionID:        params.ExecutionID,
			DatabaseID:         params.DatabaseID,
			TargetDatabaseName: targetDatabaseName,
			Status:             "running",
		})
		if err != nil {
			if isUniqueViolation(err) {
				return uuid.Nil, ErrRestoreAlreadyRunning
			}
			logError(err)
			return uuid.Nil, err
		}
		res = created
	}

	logTail := logtail.New(func(lines []string) {
		_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
			ID: res.ID,
			LogTail: sql.NullString{
				Valid:  len(lines) > 0,
				String: logtail.Join(lines),
			},
		})
	})

	fail := func(restorationID uuid.UUID, err error) (uuid.UUID, error) {
		logTail.FlushNow()
		logError(err)
		_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
			ID:         restorationID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: err.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		})
		return restorationID, err
	}

	setMessage := func(restorationID uuid.UUID, message string) {
		_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
			ID:      restorationID,
			Message: sql.NullString{Valid: true, String: message},
		})
	}

	if !params.DatabaseID.Valid && params.ConnString == "" {
		return fail(res.ID, fmt.Errorf("database_id or connection_string must be provided"))
	}

	execution, err := s.executionsService.GetExecution(ctx, params.ExecutionID)
	if err != nil {
		return fail(res.ID, err)
	}

	if execution.Status != "success" || !execution.Path.Valid {
		return fail(res.ID, fmt.Errorf("backup execution must be successful"))
	}

	connString := params.ConnString
	if params.DatabaseID.Valid {
		db, err := s.databasesService.GetDatabase(ctx, params.DatabaseID.UUID)
		if err != nil {
			return fail(res.ID, err)
		}
		connString = db.DecryptedConnectionString
	}

	pgVersion, err := s.ints.PGClient.ParseVersion(execution.DatabasePgVersion)
	if err != nil {
		return fail(res.ID, err)
	}

	// Resolve backup file location before touching the target database.
	restoreFile, err := s.executionsService.GetExecutionRestoreFile(ctx, params.ExecutionID)
	if err != nil {
		return fail(res.ID, err)
	}

	// For S3 backups: confirm the file is accessible before dropping the target database.
	if !restoreFile.IsLocal {
		if _, err = s.ints.StorageClient.S3ObjectSize(
			ctx,
			restoreFile.AccessKey, restoreFile.SecretKey,
			restoreFile.Region, restoreFile.Endpoint,
			restoreFile.Bucket, restoreFile.Path,
		); err != nil {
			return fail(res.ID, fmt.Errorf("backup file not accessible in S3: %w", err))
		}
	}

	if params.Target != nil {
		setMessage(res.ID, restorePhasePreparing)
		err = s.ints.PGClient.PrepareTargetDatabase(
			ctx, pgVersion, connString, postgres.TargetPrepareParams{
				DatabaseName: params.Target.DatabaseName,
				Owner:        params.Target.Owner,
			}, logTail,
		)
		if err != nil {
			return fail(res.ID, err)
		}
	}

	err = s.ints.PGClient.Test(ctx, pgVersion, connString)
	if err != nil {
		return fail(res.ID, err)
	}

	setMessage(res.ID, restorePhaseRestoring)
	if restoreFile.IsLocal {
		err = s.ints.PGClient.RestoreFromLocalZip(
			ctx, pgVersion, connString, restoreFile.Path, logTail,
			s.env.PBW_RESTORE_PARALLEL_JOBS,
		)
	} else {
		err = s.ints.PGClient.RestoreFromS3Zip(
			ctx,
			s.ints.StorageClient,
			pgVersion,
			connString,
			restoreFile.AccessKey,
			restoreFile.SecretKey,
			restoreFile.Region,
			restoreFile.Endpoint,
			restoreFile.Bucket,
			restoreFile.Path,
			logTail,
			s.env.PBW_RESTORE_PARALLEL_JOBS,
		)
	}
	if err != nil {
		return fail(res.ID, err)
	}

	if params.Target != nil && params.Target.Owner != "" {
		setMessage(res.ID, restorePhaseOwner)
		err = s.ints.PGClient.ReassignDatabaseOwner(ctx, pgVersion, connString, params.Target.Owner, logTail)
		if err != nil {
			return fail(res.ID, err)
		}
	}

	logger.Info("backup restored successfully", logger.KV{
		"restoration_id": res.ID.String(),
		"execution_id":   params.ExecutionID.String(),
	})
	logTail.FlushNow()
	err = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
		ID:         res.ID,
		Status:     sql.NullString{Valid: true, String: "success"},
		Message:    sql.NullString{Valid: true, String: "Backup restored successfully"},
		FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
	})
	if err != nil {
		logError(err)
		return res.ID, err
	}
	return res.ID, nil
}

func updateRestoration(
	ctx context.Context, s *Service, params dbgen.RestorationsServiceUpdateRestorationParams,
) error {
	_, err := s.dbgen.RestorationsServiceUpdateRestoration(ctx, params)
	return err
}
