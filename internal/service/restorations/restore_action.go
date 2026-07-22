package restorations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/lib/pq"
)

// RestoreStartParams selects backup and target environment.
type RestoreStartParams struct {
	Environment string
	ExecutionID *uuid.UUID
	FinishedAt  *time.Time
}

// RestoreStartResult is returned when a restore job is accepted.
type RestoreStartResult struct {
	RestorationID      uuid.UUID         `json:"restoration_id"`
	ID                 string            `json:"id"`
	Environment        string            `json:"environment"`
	Title              string            `json:"title"`
	BackupMode         string            `json:"backup_mode"`
	RestoreRequest     string            `json:"restore_request"`
	ActionRequestBody  map[string]string `json:"action_request_body,omitempty"`
	ExecutionID        uuid.UUID         `json:"execution_id"`
	BackupFinishedAt   time.Time         `json:"backup_finished_at"`
	TargetDatabaseID   *uuid.UUID        `json:"target_database_id,omitempty"`
	TargetDatabaseName string            `json:"target_database_name,omitempty"`
	Duration           string            `json:"duration,omitempty"`
	Status             string            `json:"status"`
	StatusRequest      string            `json:"status_request"`
	Message            string            `json:"message"`
}

var (
	ErrRestoreActionNotReady        = errors.New("restore is not ready for this environment")
	ErrRestoreBackupNotFound        = errors.New("backup not found for this database source")
	ErrRestoreEnvironmentNotFound   = errors.New("restore environment not found")
	ErrRestoreEnvironmentRequired   = errors.New("environment is required")
	ErrRestoreAlreadyRunning        = errors.New("restore is already running for this target database")
	ErrRestoreTargetPreflightFailed = errors.New("target preflight check failed")
)

func (s *Service) StartRestore(
	ctx context.Context,
	databaseID string,
	params RestoreStartParams,
) (RestoreStartResult, error) {
	preset, ok := findRestorePreset(databaseID)
	if !ok {
		return RestoreStartResult{}, sql.ErrNoRows
	}

	environment := normalizeEnvironment(params.Environment)
	if environment == "" {
		return RestoreStartResult{}, ErrRestoreEnvironmentRequired
	}

	target, ok := findRestoreTargetByEnvironment(preset, environment)
	if !ok {
		return RestoreStartResult{}, ErrRestoreEnvironmentNotFound
	}

	databaseIDs, err := s.loadRegisteredDatabaseIDs(ctx)
	if err != nil {
		return RestoreStartResult{}, err
	}

	buildCtx, err := s.loadPresetBuildContext(ctx, preset)
	if err != nil {
		return RestoreStartResult{}, err
	}
	if !targetActionAvailable(buildCtx) {
		return RestoreStartResult{}, ErrRestoreActionNotReady
	}

	targetConn, err := s.resolveRestoreTargetConnection(ctx, preset, target, databaseIDs)
	if err != nil {
		return RestoreStartResult{}, err
	}

	execution, backupMode, err := s.resolveRestoreExecution(ctx, preset.Source.PbwName, RestoreActionStartParams{
		ExecutionID: params.ExecutionID,
		FinishedAt:  params.FinishedAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RestoreStartResult{}, ErrRestoreBackupNotFound
		}
		return RestoreStartResult{}, err
	}
	if !execution.FinishedAt.Valid {
		return RestoreStartResult{}, fmt.Errorf("backup execution has no finished_at")
	}

	pgVersion, err := s.ints.PGClient.ParseVersion(execution.DatabasePgVersion)
	if err != nil {
		return RestoreStartResult{}, err
	}
	if err := s.ints.PGClient.PreflightTarget(ctx, pgVersion, targetConn.ConnString, target.Owner); err != nil {
		return RestoreStartResult{}, fmt.Errorf("%w: %v", ErrRestoreTargetPreflightFailed, err)
	}

	title := restoreEnvironmentTitleLatest(preset, target)
	resultBody := restoreRequestBody(environment, "")
	if backupMode == backupModeDated {
		title = restoreEnvironmentTitleDated(preset, target, execution.FinishedAt.Time)
		resultBody = restoreRequestBody(environment, execution.ID.String())
	}

	res, err := s.EnqueueRestoration(ctx, EnqueueRestorationParams{
		ExecutionID:   execution.ID,
		DatabaseID:    targetConn.DatabaseID,
		TargetPbwName: target.PbwName,
		ConnString:    targetConn.ConnString,
		Target: &RestoreTargetOptions{
			DatabaseName: target.Database,
			Owner:        target.Owner,
		},
		Tag: target.Tag,
	})
	if err != nil {
		return RestoreStartResult{}, err
	}

	restorationID := res.ID

	result := RestoreStartResult{
		RestorationID:      restorationID,
		ID:                 preset.ID,
		Environment:        environment,
		Title:              title,
		BackupMode:         backupMode,
		RestoreRequest:     restorePostRequest(preset.ID),
		ActionRequestBody:  resultBody,
		ExecutionID:        execution.ID,
		BackupFinishedAt:   execution.FinishedAt.Time,
		TargetDatabaseName: target.PbwName,
		Status:             res.Status,
		StatusRequest:      restorationStatusRequest(restorationID),
		Message:            "restore queued",
	}
	result.Duration = RestorationDuration(res.StartedAt, sql.NullTime{})
	if targetConn.DatabaseID.Valid {
		id := targetConn.DatabaseID.UUID
		result.TargetDatabaseID = &id
	}
	return result, nil
}

func (s *Service) resolveRestoreExecution(
	ctx context.Context,
	backupName string,
	params RestoreActionStartParams,
) (dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow, string, error) {
	if params.ExecutionID != nil {
		row, err := s.dbgen.RestorationsServiceGetSuccessExecutionByBackupNameAndID(
			ctx,
			dbgen.RestorationsServiceGetSuccessExecutionByBackupNameAndIDParams{
				SourceDatabaseName: backupName,
				ExecutionID:        *params.ExecutionID,
			},
		)
		if err != nil {
			return dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow{}, "", err
		}
		return latestRowFromSuccessRow(row), backupModeDated, nil
	}

	if params.FinishedAt != nil {
		dayStart, dayEnd := utcDayBounds(*params.FinishedAt)
		row, err := s.dbgen.RestorationsServiceGetSuccessExecutionByBackupNameOnDate(
			ctx,
			dbgen.RestorationsServiceGetSuccessExecutionByBackupNameOnDateParams{
				SourceDatabaseName: backupName,
				DayStart:           sql.NullTime{Valid: true, Time: dayStart},
				DayEnd:             sql.NullTime{Valid: true, Time: dayEnd},
			},
		)
		if err != nil {
			return dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow{}, "", err
		}
		return latestRowFromDateRow(row), backupModeDated, nil
	}

	row, err := s.dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupName(ctx, backupName)
	if err != nil {
		return dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow{}, "", err
	}
	return row, backupModeLatest, nil
}

func latestRowFromSuccessRow(
	row dbgen.RestorationsServiceGetSuccessExecutionByBackupNameAndIDRow,
) dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow {
	return dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow(row)
}

func latestRowFromDateRow(
	row dbgen.RestorationsServiceGetSuccessExecutionByBackupNameOnDateRow,
) dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow {
	return dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupNameRow(row)
}

func utcDayBounds(t time.Time) (time.Time, time.Time) {
	utc := t.UTC()
	dayStart := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	return dayStart, dayStart.Add(24 * time.Hour)
}

// ParseRestoreFinishedAt accepts RFC3339 datetime or YYYY-MM-DD date.
func ParseRestoreFinishedAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("finished_at is empty")
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}

	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("finished_at must be RFC3339 or YYYY-MM-DD")
	}
	return t, nil
}

// RestoreActionStartParams is kept for resolveRestoreExecution.
type RestoreActionStartParams struct {
	ExecutionID *uuid.UUID
	FinishedAt  *time.Time
}

// isUniqueViolation reports whether err is a PostgreSQL unique_violation,
// raised by restorations_one_running_per_target_uidx on concurrent restores.
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
