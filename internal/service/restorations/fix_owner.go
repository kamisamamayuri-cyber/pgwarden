package restorations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/logtail"
)

var ErrFixOwnerNoOwnerConfigured = errors.New("no owner configured for this target")

type FixOwnerResult struct {
	PresetTitle string
	Environment string
	Owner       string
}

func (s *Service) FixOwner(
	ctx context.Context, presetID, environment string,
) (FixOwnerResult, error) {
	preset, ok := findRestorePreset(presetID)
	if !ok {
		return FixOwnerResult{}, sql.ErrNoRows
	}

	environment = normalizeEnvironment(environment)
	if environment == "" {
		return FixOwnerResult{}, ErrRestoreEnvironmentRequired
	}

	target, ok := findRestoreTargetByEnvironment(preset, environment)
	if !ok {
		return FixOwnerResult{}, ErrRestoreEnvironmentNotFound
	}
	if target.Owner == "" {
		return FixOwnerResult{}, ErrFixOwnerNoOwnerConfigured
	}

	databaseIDs, err := s.loadRegisteredDatabaseIDs(ctx)
	if err != nil {
		return FixOwnerResult{}, err
	}

	buildCtx, err := s.loadPresetBuildContext(ctx, preset)
	if err != nil {
		return FixOwnerResult{}, err
	}
	if !targetActionAvailable(buildCtx) {
		return FixOwnerResult{}, ErrRestoreActionNotReady
	}

	targetConn, err := s.resolveRestoreTargetConnection(ctx, preset, target, databaseIDs)
	if err != nil {
		return FixOwnerResult{}, err
	}

	_, err = s.EnqueueRestoration(ctx, EnqueueRestorationParams{
		Op:            restoreOpFixOwner,
		ExecutionID:   buildCtx.latest.ExecutionID,
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
		return FixOwnerResult{}, err
	}

	return FixOwnerResult{
		PresetTitle: preset.Title,
		Environment: environment,
		Owner:       target.Owner,
	}, nil
}

func (s *Service) runFixOwnerTask(
	ctx context.Context,
	claim dbgen.RestorationsServiceClaimRestorationRow,
	queued queuedRestoreParams,
) error {
	logTail := logtail.New(func(lines []string) {
		_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
			ID: claim.ID,
			LogTail: sql.NullString{
				Valid:  len(lines) > 0,
				String: logtail.Join(lines),
			},
		})
	})

	fail := func(err error) error {
		logTail.FlushNow()
		_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
			ID:         claim.ID,
			Status:     sql.NullString{Valid: true, String: "failed"},
			Message:    sql.NullString{Valid: true, String: err.Error()},
			FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
		})
		return err
	}

	if queued.TargetOwner == "" {
		return fail(fmt.Errorf("fix owner task without target owner"))
	}

	connString := claim.ConnString
	if claim.DatabaseID.Valid {
		db, err := s.databasesService.GetDatabase(ctx, claim.DatabaseID.UUID)
		if err != nil {
			return fail(err)
		}
		connString = db.DecryptedConnectionString
	}
	if connString == "" {
		return fail(fmt.Errorf("fix owner task without connection string"))
	}

	execution, err := s.executionsService.GetExecution(ctx, claim.ExecutionID)
	if err != nil {
		return fail(err)
	}
	pgVersion, err := s.ints.PGClient.ParseVersion(execution.DatabasePgVersion)
	if err != nil {
		return fail(err)
	}

	_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
		ID:      claim.ID,
		Message: sql.NullString{Valid: true, String: restorePhaseOwner},
	})

	if err := s.ints.PGClient.ReassignDatabaseOwner(
		ctx, pgVersion, connString, queued.TargetOwner, logTail,
	); err != nil {
		return fail(fmt.Errorf("reassign owner: %w", err))
	}

	logTail.FlushNow()
	_ = updateRestoration(ctx, s, dbgen.RestorationsServiceUpdateRestorationParams{
		ID:         claim.ID,
		Status:     sql.NullString{Valid: true, String: "success"},
		Message:    sql.NullString{Valid: true, String: "Owner reassigned successfully"},
		FinishedAt: sql.NullTime{Valid: true, Time: time.Now()},
	})
	return nil
}
