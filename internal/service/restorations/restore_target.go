package restorations

import (
	"context"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/integration/postgres"
)

type restoreTargetConnection struct {
	DatabaseID uuid.NullUUID
	ConnString string
}

func (s *Service) resolveRestoreTargetConnection(
	ctx context.Context,
	preset RestorePreset,
	target RestoreTarget,
	databaseIDs map[string]uuid.UUID,
) (restoreTargetConnection, error) {
	if id, ok := databaseIDs[target.PbwName]; ok {
		db, err := s.databasesService.GetDatabase(ctx, id)
		if err != nil {
			return restoreTargetConnection{}, err
		}
		return restoreTargetConnection{
			DatabaseID: uuid.NullUUID{Valid: true, UUID: id},
			ConnString: db.DecryptedConnectionString,
		}, nil
	}

	sourceID, ok := databaseIDs[preset.Source.PbwName]
	if !ok {
		return restoreTargetConnection{}, ErrRestoreActionNotReady
	}

	sourceDB, err := s.databasesService.GetDatabase(ctx, sourceID)
	if err != nil {
		return restoreTargetConnection{}, err
	}

	info, err := postgres.ParseConnString(sourceDB.DecryptedConnectionString)
	if err != nil {
		return restoreTargetConnection{}, err
	}

	return restoreTargetConnection{
		ConnString: info.WithEndpoint(target.Host, target.Port, target.Database),
	}, nil
}

func targetActionAvailable(buildCtx presetBuildContext) bool {
	return buildCtx.sourceReady
}
