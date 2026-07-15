package restorations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
)

const defaultRestoreBackupsLimit = 100

// RestoreBackupListItem is one successful backup.
type RestoreBackupListItem struct {
	ExecutionID     uuid.UUID `json:"execution_id"`
	FinishedAt      time.Time `json:"finished_at"`
	Path            string    `json:"path"`
	FileSize        int64     `json:"file_size,omitempty"`
	PgVersion       string    `json:"pg_version"`
	DownloadRequest string    `json:"download_request"`
}

// RestoreBackupDownloadInfo is GET /api/v1/restores/:id/backups/:execution_id.
type RestoreBackupDownloadInfo struct {
	ID              string    `json:"id"`
	ExecutionID     uuid.UUID `json:"execution_id"`
	DownloadRequest string    `json:"download_request"`
	IsLocal         bool      `json:"is_local"`
	DownloadURL     string    `json:"download_url,omitempty"`
}

// RestoreBackupList is GET /api/v1/restores/:id/backups.
type RestoreBackupList struct {
	ID                 string                `json:"id"`
	Title              string                `json:"title"`
	Source             RestoreEndpoint       `json:"source"`
	BackupsListRequest string                `json:"backups_list_request"`
	RestoreListRequest string                `json:"restore_list_request"`
	RestoreRequest     string                `json:"restore_request"`
	Backups            []RestoreBackupListItem `json:"backups"`
}

func (s *Service) ListRestoreBackups(
	ctx context.Context, id string, limit int,
) (RestoreBackupList, error) {
	preset, ok := findRestorePreset(id)
	if !ok {
		return RestoreBackupList{}, sql.ErrNoRows
	}

	if limit < 1 {
		limit = defaultRestoreBackupsLimit
	}

	rows, err := s.dbgen.RestorationsServiceListSuccessExecutionsByBackupName(
		ctx,
		dbgen.RestorationsServiceListSuccessExecutionsByBackupNameParams{
			SourceDatabaseName: preset.Source.PbwName,
			Limit:              int32(limit),
		},
	)
	if err != nil {
		return RestoreBackupList{}, err
	}

	backups := make([]RestoreBackupListItem, 0, len(rows))
	for _, row := range rows {
		if !row.FinishedAt.Valid || !row.Path.Valid {
			continue
		}

		item := RestoreBackupListItem{
			ExecutionID:     row.ID,
			FinishedAt:      row.FinishedAt.Time,
			Path:            row.Path.String,
			PgVersion:       row.DatabasePgVersion,
			DownloadRequest: restoreBackupDownloadRequest(preset.ID, row.ID.String()),
		}
		if row.FileSize.Valid {
			item.FileSize = row.FileSize.Int64
		}

		backups = append(backups, item)
	}

	return RestoreBackupList{
		ID:                 preset.ID,
		Title:              preset.Title,
		Source:             preset.Source,
		BackupsListRequest: restoreBackupsListRequest(preset.ID),
		RestoreListRequest: restoreTargetsListRequest(preset.ID),
		RestoreRequest:     restorePostRequest(preset.ID),
		Backups:            backups,
	}, nil
}

func (s *Service) GetRestoreBackupDownload(
	ctx context.Context, databaseID string, executionID uuid.UUID,
) (RestoreBackupDownloadInfo, error) {
	preset, ok := findRestorePreset(databaseID)
	if !ok {
		return RestoreBackupDownloadInfo{}, sql.ErrNoRows
	}

	row, err := s.dbgen.RestorationsServiceGetSuccessExecutionByBackupNameAndID(
		ctx,
		dbgen.RestorationsServiceGetSuccessExecutionByBackupNameAndIDParams{
			SourceDatabaseName: preset.Source.PbwName,
			ExecutionID:        executionID,
		},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RestoreBackupDownloadInfo{}, sql.ErrNoRows
		}
		return RestoreBackupDownloadInfo{}, err
	}
	if !row.Path.Valid {
		return RestoreBackupDownloadInfo{}, fmt.Errorf("backup has no file")
	}

	isLocal, downloadURL, err := s.executionsService.GetExecutionDownloadLinkOrPath(
		ctx, executionID,
	)
	if err != nil {
		return RestoreBackupDownloadInfo{}, err
	}

	info := RestoreBackupDownloadInfo{
		ID:              preset.ID,
		ExecutionID:     executionID,
		DownloadRequest: restoreBackupDownloadRequest(preset.ID, executionID.String()),
		IsLocal:         isLocal,
	}
	if !isLocal {
		info.DownloadURL = downloadURL
	}

	return info, nil
}
