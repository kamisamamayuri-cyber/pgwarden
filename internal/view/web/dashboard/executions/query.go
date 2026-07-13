package executions

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
	"github.com/google/uuid"
)

type execsFilterQuery struct {
	Database    uuid.UUID
	Destination uuid.UUID
	Backup      uuid.UUID
	Status      string
}

func buildExecutionsIndexURL(q execsFilterQuery, status string) string {
	url := pathutil.BuildPath("/dashboard/executions")
	if q.Database != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "database", q.Database.String())
	}
	if q.Destination != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "destination", q.Destination.String())
	}
	if q.Backup != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "backup", q.Backup.String())
	}
	if status != "" {
		url = strutil.AddQueryParamToUrl(url, "status", status)
	}
	return url
}

func buildExecutionsListURL(q execsFilterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/executions/list")
	url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", page))
	if q.Database != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "database", q.Database.String())
	}
	if q.Destination != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "destination", q.Destination.String())
	}
	if q.Backup != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "backup", q.Backup.String())
	}
	if q.Status != "" {
		url = strutil.AddQueryParamToUrl(url, "status", q.Status)
	}
	return url
}
