package restorations

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// LatestBackupInfo is the newest successful backup for a preset source.
type LatestBackupInfo struct {
	ExecutionID uuid.UUID `json:"execution_id"`
	FinishedAt  time.Time `json:"finished_at"`
	Path        string    `json:"path"`
	FileSize    int64     `json:"file_size,omitempty"`
	PgVersion   string    `json:"pg_version"`
}

// RestoreDatabaseInfo is GET /api/v1/restores/:id.
type RestoreDatabaseInfo struct {
	ID                 string            `json:"id"`
	Title              string            `json:"title"`
	Description        string            `json:"description,omitempty"`
	Source             RestoreEndpoint   `json:"source"`
	LatestBackup       *LatestBackupInfo `json:"latest_backup,omitempty"`
	BackupsListRequest string            `json:"backups_list_request"`
	RestoreListRequest string            `json:"restore_list_request"`
	RestoreRequest     string            `json:"restore_request"`
	ActionAvailable    bool              `json:"action_available"`
	CanExecute         bool              `json:"can_execute"`
}

// RestoreEnvironmentInfo is one restore destination (stage, rc, …).
type RestoreEnvironmentInfo struct {
	Environment                      string            `json:"environment"`
	Title                            string            `json:"title"`
	BackupMode                       string            `json:"backup_mode"`
	Target                           RestoreEndpoint   `json:"target"`
	ActionAvailable                  bool              `json:"action_available"`
	ActionRequest                    string            `json:"action_request"`
	ActionRequestBody                map[string]string `json:"action_request_body,omitempty"`
	ActionRequestBodyNote            string            `json:"action_request_body_note,omitempty"`
	ActionRequestBodyWithExecutionID map[string]string `json:"action_request_body_with_execution_id,omitempty"`
	ActionRequestBodyWithFinishedAt  map[string]string `json:"action_request_body_with_finished_at,omitempty"`
	BackupsListRequest               string            `json:"backups_list_request,omitempty"`
}

// RestoreTargetsInfo is GET /api/v1/restores/:id/restore.
type RestoreTargetsInfo struct {
	ID                 string                   `json:"id"`
	Title              string                   `json:"title"`
	Source             RestoreEndpoint          `json:"source"`
	RestoreListRequest string                   `json:"restore_list_request"`
	RestoreRequest     string                   `json:"restore_request"`
	Environments       []RestoreEnvironmentInfo `json:"environments"`
}

// RestoreCatalog is GET /api/v1/restores.
type RestoreCatalog struct {
	Databases []RestoreDatabaseInfo `json:"databases"`
}

type presetBuildContext struct {
	sourceReady bool
	latest      *LatestBackupInfo
}

func (s *Service) ListRestoreCatalog(ctx context.Context) (RestoreCatalog, error) {
	databases, err := s.ListRestoreDatabases(ctx)
	if err != nil {
		return RestoreCatalog{}, err
	}
	return RestoreCatalog{Databases: databases}, nil
}

func (s *Service) ListRestoreDatabases(ctx context.Context) ([]RestoreDatabaseInfo, error) {
	result := make([]RestoreDatabaseInfo, 0, len(getRestorePresets()))
	for _, preset := range getRestorePresets() {
		info, err := s.buildRestoreDatabaseInfo(ctx, preset)
		if err != nil {
			return nil, err
		}
		result = append(result, info)
	}
	return result, nil
}

func (s *Service) GetRestoreDatabase(
	ctx context.Context, id string,
) (RestoreDatabaseInfo, error) {
	preset, ok := findRestorePreset(id)
	if !ok {
		return RestoreDatabaseInfo{}, sql.ErrNoRows
	}

	return s.buildRestoreDatabaseInfo(ctx, preset)
}

func (s *Service) GetRestoreTargets(
	ctx context.Context, id string,
) (RestoreTargetsInfo, error) {
	preset, ok := findRestorePreset(id)
	if !ok {
		return RestoreTargetsInfo{}, sql.ErrNoRows
	}

	buildCtx, err := s.loadPresetBuildContext(ctx, preset)
	if err != nil {
		return RestoreTargetsInfo{}, err
	}

	info := RestoreTargetsInfo{
		ID:                 preset.ID,
		Title:              preset.Title,
		Source:             preset.Source,
		RestoreListRequest: restoreTargetsListRequest(preset.ID),
		RestoreRequest:     restorePostRequest(preset.ID),
		Environments:       make([]RestoreEnvironmentInfo, 0, len(preset.Targets)),
	}

	for _, target := range preset.Targets {
		envInfo := buildRestoreEnvironmentInfo(preset, target, buildCtx)
		info.Environments = append(info.Environments, envInfo)
	}

	return info, nil
}

func (s *Service) loadRegisteredDatabaseIDs(
	ctx context.Context,
) (map[string]uuid.UUID, error) {
	databases, err := s.databasesService.GetAllDatabases(ctx)
	if err != nil {
		return nil, err
	}

	databaseIDs := map[string]uuid.UUID{}
	for _, db := range databases {
		databaseIDs[db.Name] = db.ID
	}
	return databaseIDs, nil
}

func (s *Service) loadPresetBuildContext(
	ctx context.Context,
	preset RestorePreset,
) (presetBuildContext, error) {
	buildCtx := presetBuildContext{}

	latest, err := s.dbgen.RestorationsServiceGetLatestSuccessExecutionByBackupName(
		ctx, preset.Source.PbwName,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return presetBuildContext{}, err
	}
	if errors.Is(err, sql.ErrNoRows) || !latest.FinishedAt.Valid || !latest.Path.Valid {
		return buildCtx, nil
	}

	buildCtx.sourceReady = true
	buildCtx.latest = &LatestBackupInfo{
		ExecutionID: latest.ID,
		FinishedAt:  latest.FinishedAt.Time,
		Path:        latest.Path.String,
		PgVersion:   latest.DatabasePgVersion,
	}
	if latest.FileSize.Valid {
		buildCtx.latest.FileSize = latest.FileSize.Int64
	}

	return buildCtx, nil
}

func (s *Service) buildRestoreDatabaseInfo(
	ctx context.Context,
	preset RestorePreset,
) (RestoreDatabaseInfo, error) {
	buildCtx, err := s.loadPresetBuildContext(ctx, preset)
	if err != nil {
		return RestoreDatabaseInfo{}, err
	}

	info := RestoreDatabaseInfo{
		ID:                 preset.ID,
		Title:              preset.Title,
		Description:        preset.Description,
		Source:             preset.Source,
		LatestBackup:       buildCtx.latest,
		BackupsListRequest: restoreBackupsListRequest(preset.ID),
		RestoreListRequest: restoreTargetsListRequest(preset.ID),
		RestoreRequest:     restorePostRequest(preset.ID),
		ActionAvailable:    targetActionAvailable(buildCtx),
	}

	return info, nil
}

func buildRestoreEnvironmentInfo(
	preset RestorePreset,
	target RestoreTarget,
	buildCtx presetBuildContext,
) RestoreEnvironmentInfo {
	available := targetActionAvailable(buildCtx)

	return RestoreEnvironmentInfo{
		Environment:                      target.Environment,
		Title:                            restoreEnvironmentTitleLatest(preset, target),
		BackupMode:                       backupModeLatest,
		Target:                           target.RestoreEndpoint,
		ActionAvailable:                  available,
		ActionRequest:                    restorePostRequest(preset.ID),
		ActionRequestBody:                restoreRequestBody(target.Environment, ""),
		ActionRequestBodyNote:            restoreActionRequestBodyNote(preset.ID),
		ActionRequestBodyWithExecutionID: restoreRequestBody(target.Environment, "<execution_id>"),
		ActionRequestBodyWithFinishedAt: restoreRequestBodyWithFinishedAtDate(
			target.Environment, "2025-06-15",
		),
		BackupsListRequest: restoreBackupsListRequest(preset.ID),
	}
}
