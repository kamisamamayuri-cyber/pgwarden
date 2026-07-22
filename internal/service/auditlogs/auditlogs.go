package auditlogs

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

const auditLogPurgeBatchLimit = 500

type Service struct {
	db            dbgen.DBTX
	retentionDays int
}

func New(db dbgen.DBTX, retentionDays int) *Service {
	return &Service{db: db, retentionDays: retentionDays}
}

type Entry struct {
	UserEmail   string
	Action      string
	PresetID    string
	PresetTitle string
	ExecutionID *uuid.UUID
	Environment *string
	Source      string
}

type LogRow struct {
	ID          uuid.UUID
	CreatedAt   time.Time
	UserEmail   string
	Action      string
	PresetID    string
	PresetTitle string
	ExecutionID *uuid.UUID
	Environment *string
	Source      string
}

func (s *Service) Log(ctx context.Context, e Entry) {
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO audit_logs
			(user_email, action, preset_id, preset_title, execution_id, environment, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		e.UserEmail, e.Action, e.PresetID, e.PresetTitle, e.ExecutionID, e.Environment, e.Source,
	)
}

func (s *Service) List(ctx context.Context, page, limit int) ([]LogRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, user_email, action, preset_id, preset_title,
		       execution_id, environment, source
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LogRow
	for rows.Next() {
		var r LogRow
		var execID sql.NullString
		var env sql.NullString
		if err := rows.Scan(
			&r.ID, &r.CreatedAt, &r.UserEmail, &r.Action,
			&r.PresetID, &r.PresetTitle, &execID, &env, &r.Source,
		); err != nil {
			return nil, err
		}
		if execID.Valid {
			id, _ := uuid.Parse(execID.String)
			r.ExecutionID = &id
		}
		if env.Valid {
			r.Environment = &env.String
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// HousekeepAuditLogs purges audit log entries older than retentionDays.
func (s *Service) HousekeepAuditLogs() {
	ctx := context.Background()

	res, err := s.db.ExecContext(ctx, `
		DELETE FROM audit_logs
		WHERE id IN (
			SELECT id FROM audit_logs
			WHERE created_at < NOW() - ($1 || ' days')::INTERVAL
			ORDER BY created_at ASC
			LIMIT $2
		)`,
		s.retentionDays, auditLogPurgeBatchLimit,
	)
	if err != nil {
		logger.Error("error purging old audit logs", logger.KV{"error": err})
		return
	}

	purged, err := res.RowsAffected()
	if err != nil || purged == 0 {
		return
	}

	logger.Info("old audit log entries purged", logger.KV{"purged": purged})
}
