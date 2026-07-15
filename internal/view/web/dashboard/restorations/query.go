package restorations

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/strutil"
	"github.com/google/uuid"
)

type resFilterQuery struct {
	Execution uuid.UUID
	Database  uuid.UUID
	Status    string
}

func buildRestorationsIndexURL(q resFilterQuery, status string) string {
	url := pathutil.BuildPath("/dashboard/restorations")
	if q.Execution != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "execution", q.Execution.String())
	}
	if q.Database != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "database", q.Database.String())
	}
	if status != "" {
		url = strutil.AddQueryParamToUrl(url, "status", status)
	}
	return url
}

func buildRestorationDetailsURL(restorationID uuid.UUID) string {
	return pathutil.BuildPath(
		fmt.Sprintf("/dashboard/restorations/%s/details", restorationID),
	)
}

func buildRestorationsListURL(q resFilterQuery, page int) string {
	url := pathutil.BuildPath("/dashboard/restorations/list")
	url = strutil.AddQueryParamToUrl(url, "page", fmt.Sprintf("%d", page))
	if q.Execution != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "execution", q.Execution.String())
	}
	if q.Database != uuid.Nil {
		url = strutil.AddQueryParamToUrl(url, "database", q.Database.String())
	}
	if q.Status != "" {
		url = strutil.AddQueryParamToUrl(url, "status", q.Status)
	}
	return url
}
