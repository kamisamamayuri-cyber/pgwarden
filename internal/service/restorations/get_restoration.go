package restorations

import (
	"context"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/google/uuid"
)

func (s *Service) GetRestorationRow(
	ctx context.Context, restorationID uuid.UUID,
) (dbgen.RestorationsServiceGetRestorationRow, error) {
	return s.dbgen.RestorationsServiceGetRestoration(ctx, restorationID)
}

// RestorationStatusInfo is GET /api/v1/restorations/:restoration_id.
type RestorationStatusInfo struct {
	RestorationID uuid.UUID  `json:"restoration_id"`
	Status        string     `json:"status"`
	Message       string     `json:"message,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
	Duration           string     `json:"duration,omitempty"`
	ExecutionID        uuid.UUID  `json:"execution_id"`
	DatabaseID         *uuid.UUID `json:"database_id,omitempty"`
	DatabaseName       string     `json:"database_name,omitempty"`
	TargetDatabaseName string     `json:"target_database_name,omitempty"`
	BackupName         string     `json:"backup_name,omitempty"`
	LogLines           []string   `json:"log_lines,omitempty"`
	StatusRequest      string     `json:"status_request"`
}

func (s *Service) GetRestorationStatus(
	ctx context.Context, restorationID uuid.UUID,
) (RestorationStatusInfo, error) {
	row, err := s.dbgen.RestorationsServiceGetRestoration(ctx, restorationID)
	if err != nil {
		return RestorationStatusInfo{}, err
	}

	info := RestorationStatusInfo{
		RestorationID: row.ID,
		Status:        row.Status,
		StartedAt:     row.StartedAt,
		ExecutionID:   row.ExecutionID,
		StatusRequest: restorationStatusRequest(row.ID),
	}
	if row.Message.Valid {
		info.Message = row.Message.String
	}
	if row.UpdatedAt.Valid {
		t := row.UpdatedAt.Time
		info.UpdatedAt = &t
	}
	if row.FinishedAt.Valid {
		t := row.FinishedAt.Time
		info.FinishedAt = &t
	}
	if row.DatabaseID.Valid {
		id := row.DatabaseID.UUID
		info.DatabaseID = &id
	}
	if row.DatabaseName.Valid {
		info.DatabaseName = row.DatabaseName.String
	}
	if row.TargetDatabaseName.Valid && row.TargetDatabaseName.String != "" {
		info.TargetDatabaseName = row.TargetDatabaseName.String
	} else if info.DatabaseName != "" {
		info.TargetDatabaseName = info.DatabaseName
	}
	if row.BackupName != "" {
		info.BackupName = row.BackupName
	}
	if row.LogTail.Valid {
		info.LogLines = LogTailToLines(row.LogTail.String)
	}
	info.Duration = RestorationDuration(row.StartedAt, row.FinishedAt)

	return info, nil
}
