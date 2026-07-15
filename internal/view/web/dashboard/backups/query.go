package backups

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
)

type backupsFilterQuery struct {
	Host string
}

func buildBackupsIndexURL(host string) string {
	url := pathutil.BuildPath("/dashboard/backups")
	if host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", host)
	}
	return url
}

func buildBackupsListURL(q backupsFilterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/backups/list")
	url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", page))
	if q.Host != "" {
		url = strutil.AddQueryParamToUrl(url, "host", q.Host)
	}
	return url
}
