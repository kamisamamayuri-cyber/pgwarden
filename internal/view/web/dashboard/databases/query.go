package databases

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
)

type databasesFilterQuery struct {
	Host string
}

func buildDatabasesIndexURL(host string) string {
	url := pathutil.BuildPath("/dashboard/databases")
	if host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", host)
	}
	return url
}

func buildDatabasesListURL(q databasesFilterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/databases/list")
	url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", page))
	if q.Host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", q.Host)
	}
	return url
}
