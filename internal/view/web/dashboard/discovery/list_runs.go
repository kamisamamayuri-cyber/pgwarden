package discovery

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
)

func (h *handlers) listRunsHandler(c echo.Context) error {
	var queryData listQueryData
	if err := c.Bind(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	q := filterQuery{
		Level:    queryData.Level,
		Event:    queryData.Event,
		Host:     queryData.Host,
		Port:     queryData.Port,
		Database: queryData.Database,
	}
	pagination, runs, err := h.servs.DiscoveryService.PaginateRuns(
		c.Request().Context(),
		paginateParamsFromQuery(queryData),
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, listRuns(q, pagination, runs))
}

func listRuns(
	q filterQuery,
	pagination paginateutil.PaginateResponse,
	runs []dbgen.DiscoveryServicePaginateRunsRow,
) nodx.Node {
	if len(runs) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No discovery runs found",
			Subtitle: "Discovery runs will appear here",
		})
	}

	trs := make([]nodx.Node, 0, len(runs)+1)
	for _, run := range runs {
		trs = append(trs, discoveryRunRow(run))
	}

	if pagination.HasNextPage {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(buildListURL(q, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}

func discoveryRunRow(run dbgen.DiscoveryServicePaginateRunsRow) nodx.Node {
	status := "running"
	switch {
	case run.Finished && run.ErrorsCount == 0:
		status = "success"
	case run.Finished && run.ErrorsCount > 0:
		status = "failed"
	case isDiscoveryRunStale(run):
		status = "failed"
	}

	return nodx.Tr(
		nodx.Td(discoveryRunActions(run)),
		nodx.Td(component.SpanText(
			discoveryRunTime(run.StartedAt),
		)),
		nodx.Td(component.StatusBadge(status)),
		nodx.Td(component.SpanText(int32Text(run.PortsCount))),
		nodx.Td(component.SpanText(int32Text(run.DatabasesCount))),
		nodx.Td(component.SpanText(int32Text(run.DatabasesCreatedCount))),
		nodx.Td(component.SpanText(int32Text(run.BackupsCreatedCount))),
		nodx.Td(component.SpanText(int32Text(run.SkippedExistingCount))),
		nodx.Td(component.SpanText(int32Text(run.ErrorsCount))),
		nodx.Td(component.SpanText(
			discoveryRunTime(run.UpdatedAt),
		)),
	)
}

func discoveryRunActions(run dbgen.DiscoveryServicePaginateRunsRow) nodx.Node {
	runID := run.RunID.String()
	logModalID := "discovery-run-log-" + runID
	reportModalID := "discovery-run-report-" + runID

	logModal := component.Modal(component.ModalParams{
		ID:    logModalID,
		Title: "Discovery log",
		Size:  component.SizeXl,
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(buildRunDetailsURL(runID)),
				htmx.HxTrigger(logModalID+"_open from:window"),
				htmx.HxSwap("outerHTML"),
				component.SkeletonTr(7),
			),
		},
	})
	reportModal := component.Modal(component.ModalParams{
		ID:    reportModalID,
		Title: "Discovery report",
		Size:  component.SizeLg,
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(buildRunReportURL(runID)),
				htmx.HxTrigger(reportModalID+"_open from:window"),
				htmx.HxSwap("outerHTML"),
				component.SkeletonTr(7),
			),
		},
	})

	return nodx.Div(
		nodx.Class("flex gap-1"),
		logModal.HTML,
		reportModal.HTML,
		nodx.Div(
			nodx.Class("tooltip"),
			nodx.Data("tip", "Log"),
			nodx.Button(
				logModal.OpenerAttr,
				nodx.Class("btn btn-square btn-sm btn-ghost"),
				lucide.Eye(),
			),
		),
		nodx.Div(
			nodx.Class("tooltip"),
			nodx.Data("tip", "Report"),
			nodx.Button(
				reportModal.OpenerAttr,
				nodx.Class("btn btn-square btn-sm btn-ghost"),
				lucide.FileText(),
			),
		),
	)
}

func int32Text(value int32) string {
	return strconv.Itoa(int(value))
}

func discoveryRunTime(value any) string {
	t, ok := value.(time.Time)
	if !ok {
		return ""
	}
	return t.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty)
}

func isDiscoveryRunStale(run dbgen.DiscoveryServicePaginateRunsRow) bool {
	if run.Finished {
		return false
	}
	updatedAt, ok := run.UpdatedAt.(time.Time)
	if !ok {
		return false
	}
	return time.Since(updatedAt) > 30*time.Minute
}
