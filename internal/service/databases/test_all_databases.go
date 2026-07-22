package databases

import (
	"context"
	"slices"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"golang.org/x/sync/errgroup"
)

func filterByTags(
	databases []dbgen.DatabasesServiceGetAllDatabasesRow, tags []string,
) []dbgen.DatabasesServiceGetAllDatabasesRow {
	if len(tags) == 0 {
		return databases
	}
	out := make([]dbgen.DatabasesServiceGetAllDatabasesRow, 0, len(databases))
	for _, db := range databases {
		if slices.Contains(tags, db.Tag) {
			out = append(out, db)
		}
	}
	return out
}

func (s *Service) TestAllDatabases(tags []string) {
	ctx := context.Background()

	databases, err := s.GetAllDatabases(ctx)
	if err != nil {
		logger.Error(
			"error getting all databases to test them", logger.KV{"error": err},
		)
		return
	}

	databases = filterByTags(databases, tags)

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(5)

	for _, db := range databases {
		db := db
		eg.Go(func() error {
			err := s.TestDatabaseAndStoreResult(ctx, db.ID)
			if err != nil {
				logger.Error(
					"error testing database",
					logger.KV{"database_id": db.ID, "error": err},
				)
			}
			return nil
		})
	}

	_ = eg.Wait()
	logger.Info("all databases tested")

}
