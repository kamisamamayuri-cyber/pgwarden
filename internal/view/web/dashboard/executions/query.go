package executions

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
)

type execsFilterQuery struct {
	Database    uuid.UUID
	Destination uuid.UUID
	Backup      uuid.UUID
	Status      string
	Type        string
	Host        string
}

func buildExecutionsIndexURL(q execsFilterQuery, status, typ string) string {
	url := pathutil.BuildPath("/dashboard/jobs")
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
	if typ != "" {
		url = strutil.AddQueryParamToUrl(url, "type", typ)
	}
	if q.Host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", q.Host)
	}
	return url
}

func buildExecutionsListURL(q execsFilterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/jobs/list")
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
	if q.Type != "" {
		url = strutil.AddQueryParamToUrl(url, "type", q.Type)
	}
	if q.Host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", q.Host)
	}
	return url
}

func buildExecutionsListPollURL(q execsFilterQuery, page int) string {
	return strutil.AddQueryParamToUrl(buildExecutionsListURL(q, page), "poll", "1")
}
