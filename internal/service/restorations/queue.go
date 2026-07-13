package restorations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
)

// queuedRestoreParams is the non-secret part of a queued restoration, stored
// as JSON in restorations.params. The connection string is stored separately,
// encrypted with pgcrypto (restorations.enc_conn_string).
type queuedRestoreParams struct {
	TargetDatabaseName string `json:"target_database_name,omitempty"`
	TargetOwner        string `json:"target_owner,omitempty"`
}

// EnqueueRestorationParams describes a restore job to put into the queue.
type EnqueueRestorationParams struct {
	ExecutionID   uuid.UUID
	DatabaseID    uuid.NullUUID
	TargetPbwName string
	// ConnString is required when DatabaseID is not set; stored encrypted.
	ConnString string
	Target     *RestoreTargetOptions
}

// EnqueueRestoration puts a restore job into the queue. A worker picks it up
// via ClaimRestoration. The partial unique index
// restorations_one_active_per_target_uidx rejects a second queued/running
// restore of the same target database.
func (s *Service) EnqueueRestoration(
	ctx context.Context, params EnqueueRestorationParams,
) (dbgen.Restoration, error) {
	// Best-effort target name for display and the one-active-per-target
	// uniqueness guard (mirrors the old RunRestoration behaviour).
	if params.TargetPbwName == "" && params.DatabaseID.Valid {
		if db, err := s.databasesService.GetDatabase(ctx, params.DatabaseID.UUID); err == nil {
			params.TargetPbwName = db.Name
		}
	}

	queued := queuedRestoreParams{}
	if params.Target != nil {
		queued.TargetDatabaseName = params.Target.DatabaseName
		queued.TargetOwner = params.Target.Owner
	}
	rawParams, err := json.Marshal(queued)
	if err != nil {
		return dbgen.Restoration{}, err
	}

	res, err := s.dbgen.RestorationsServiceEnqueueRestoration(
		ctx, dbgen.RestorationsServiceEnqueueRestorationParams{
			ExecutionID: params.ExecutionID,
			DatabaseID:  params.DatabaseID,
			TargetDatabaseName: sql.NullString{
				Valid:  params.TargetPbwName != "",
				String: params.TargetPbwName,
			},
			Params:        sql.NullString{Valid: true, String: string(rawParams)},
			ConnString:    params.ConnString,
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if err != nil {
		if isUniqueViolation(err) {
			return dbgen.Restoration{}, ErrRestoreAlreadyRunning
		}
		return dbgen.Restoration{}, err
	}
	return res, nil
}

// ClaimRestoration atomically claims one queued restoration for a worker.
// Returns ok=false when the queue is empty.
func (s *Service) ClaimRestoration(
	ctx context.Context, claimedBy string,
) (dbgen.RestorationsServiceClaimRestorationRow, bool, error) {
	row, err := s.dbgen.RestorationsServiceClaimRestoration(
		ctx, dbgen.RestorationsServiceClaimRestorationParams{
			ClaimedBy:     sql.NullString{Valid: true, String: claimedBy},
			EncryptionKey: s.env.PBW_ENCRYPTION_KEY,
		},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return row, false, nil
	}
	if err != nil {
		return row, false, err
	}
	return row, true, nil
}

// HeartbeatRestoration refreshes the liveness timestamp of a running restore.
func (s *Service) HeartbeatRestoration(ctx context.Context, id uuid.UUID) error {
	return s.dbgen.RestorationsServiceHeartbeatRestoration(ctx, id)
}

// ReapStaleRestorations fails running restorations whose worker stopped
// heartbeating (crashed or was killed without cleanup).
func (s *Service) ReapStaleRestorations(
	ctx context.Context, staleAfter time.Duration,
) (int64, error) {
	return s.dbgen.RestorationsServiceReapStaleRestorations(
		ctx, int32(staleAfter/time.Second),
	)
}

// RunClaimedRestoration executes a restoration claimed from the queue.
func (s *Service) RunClaimedRestoration(
	ctx context.Context, claim dbgen.RestorationsServiceClaimRestorationRow,
) error {
	queued := queuedRestoreParams{}
	if claim.Params.Valid && claim.Params.String != "" {
		if err := json.Unmarshal([]byte(claim.Params.String), &queued); err != nil {
			failErr := fmt.Errorf("invalid queued restore params: %w", err)
			logger.Error("restoration params unmarshal failed", logger.KV{
				"restoration_id": claim.ID.String(),
				"error":          err.Error(),
			})
			_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
				ID:         claim.ID,
				Status:     sql.NullString{Valid: true, String: "failed"},
				Message:    sql.NullString{Valid: true, String: failErr.Error()},
				FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
			})
			return failErr
		}
	}

	var target *RestoreTargetOptions
	if queued.TargetDatabaseName != "" || queued.TargetOwner != "" {
		target = &RestoreTargetOptions{
			DatabaseName: queued.TargetDatabaseName,
			Owner:        queued.TargetOwner,
		}
	}

	_, err := s.RunRestoration(ctx, RunRestorationParams{
		ExecutionID:           claim.ExecutionID,
		DatabaseID:            claim.DatabaseID,
		ConnString:            claim.ConnString,
		ExistingRestorationID: uuid.NullUUID{Valid: true, UUID: claim.ID},
		Target:                target,
	})
	return err
}
