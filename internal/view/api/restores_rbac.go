package api

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
)

func filterRestoreCatalog(
	catalog restorations.RestoreCatalog,
	canView func(string) bool,
	canExecute func(string) bool,
) restorations.RestoreCatalog {
	filtered := restorations.RestoreCatalog{
		Databases: make([]restorations.RestoreDatabaseInfo, 0, len(catalog.Databases)),
	}
	for _, item := range catalog.Databases {
		if canView(item.ID) {
			item.CanExecute = canExecute(item.ID)
			filtered.Databases = append(filtered.Databases, item)
		}
	}
	return filtered
}
