package discovery

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
)

type filterQuery struct {
	Level    string
	Event    string
	Host     string
	Port     int
	Database string
}

func buildIndexURL(q filterQuery) string {
	url := pathutil.BuildPath("/dashboard/discovery")
	return addFilterQuery(url, q)
}

func buildListURL(q filterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/discovery/list")
	url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", page))
	return addFilterQuery(url, q)
}

func buildRunURL(q filterQuery) string {
	url := pathutil.BuildPath("/dashboard/discovery/run")
	return addFilterQuery(url, q)
}

func buildRunDetailsURL(runID string) string {
	return pathutil.BuildPath(fmt.Sprintf("/dashboard/discovery/runs/%s/details", runID))
}

func buildRunReportURL(runID string) string {
	return pathutil.BuildPath(fmt.Sprintf("/dashboard/discovery/runs/%s/report", runID))
}

func addFilterQuery(url string, q filterQuery) string {
	if q.Level != "" {
		url = strutil.AddQueryParamToUrl(url, "level", q.Level)
	}
	if q.Event != "" {
		url = strutil.AddQueryParamToUrl(url, "event", q.Event)
	}
	if q.Host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", q.Host)
	}
	if q.Port > 0 {
		url = strutil.AddQueryParamToUrl(url, "port", fmt.Sprintf("%d", q.Port))
	}
	if q.Database != "" {
		url = strutil.AddQueryParamToUrl(url, "database", q.Database)
	}
	return url
}
