package discovery

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

const discoveryRunsPageSize = 30
const discoveryRunsPollMaxRows = 300
const htmxStopPollingStatus = 286

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

	isPoll := c.QueryParam("poll") == "1"
	params := paginateParamsFromQuery(queryData)
	if isPoll {
		rows, _ := strconv.Atoi(c.QueryParam("rows"))
		params.Limit = pollLimit(rows, discoveryRunsPageSize, discoveryRunsPollMaxRows)
	}

	pagination, runs, err := h.servs.DiscoveryService.PaginateRuns(
		c.Request().Context(),
		params,
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if queryData.Page > 1 {
		return echoutil.RenderNodx(c, http.StatusOK, listRuns(q, pagination, runs))
	}

	active, err := h.servs.DiscoveryService.Running(c.Request().Context())
	if err != nil {
		active = false
	}

	if isPoll {
		pagination.NextPage = params.Limit/discoveryRunsPageSize + 1
		status := http.StatusOK
		if !active {
			status = htmxStopPollingStatus
		}
		return echoutil.RenderNodx(c, status, buildRunRows(q, pagination, runs))
	}

	return echoutil.RenderNodx(c, http.StatusOK, nodx.Group(
		renderDiscoveryRunsTbody(q, pagination, runs, active),
		discoveryRunModalsOOB(runs),
	))
}

func pollLimit(rows, pageSize, maxRows int) int {
	if rows < pageSize {
		return pageSize
	}
	if rows > maxRows {
		rows = maxRows
	}
	return (rows + pageSize - 1) / pageSize * pageSize
}

const discoveryRunsTbodyID = "discovery-events"
const discoveryRunModalsContainerID = "discovery-run-modals"

func renderDiscoveryRunsTbody(
	q filterQuery,
	pagination paginateutil.PaginateResponse,
	runs []dbgen.DiscoveryServicePaginateRunsRow,
	poll bool,
) nodx.Node {
	attrs := []nodx.Node{nodx.Id(discoveryRunsTbodyID)}
	if poll {
		attrs = append(attrs,
			htmx.HxGet(buildListPollURL(q, 1)),
			htmx.HxTrigger("every 5s"),
			htmx.HxSwap("innerHTML"),
			nodx.Attr("hx-vals",
				"js:{rows: document.querySelectorAll('#"+discoveryRunsTbodyID+" > div[data-row]').length}"),
		)
	}
	return nodx.Div(append(attrs, buildRunRows(q, pagination, runs))...)
}

func listRuns(
	q filterQuery,
	pagination paginateutil.PaginateResponse,
	runs []dbgen.DiscoveryServicePaginateRunsRow,
) nodx.Node {
	return component.RenderableGroup(
		[]nodx.Node{
			buildRunRows(q, pagination, runs),
			discoveryRunModalsOOB(runs),
		},
	)
}

func discoveryRunModalsOOB(runs []dbgen.DiscoveryServicePaginateRunsRow) nodx.Node {
	if len(runs) < 1 {
		return nil
	}

	modals := make([]nodx.Node, 0, len(runs)*2)
	for _, run := range runs {
		lm, rm := discoveryRunModals(run)
		modals = append(modals, lm, rm)
	}

	return nodx.Div(
		nodx.Id(discoveryRunModalsContainerID),
		nodx.Attr("hx-swap-oob", "beforeend"),
		nodx.Group(modals...),
	)
}

func buildRunRows(
	q filterQuery,
	pagination paginateutil.PaginateResponse,
	runs []dbgen.DiscoveryServicePaginateRunsRow,
) nodx.Node {
	if len(runs) < 1 {
		return component.EmptyResults(component.EmptyResultsParams{
			Title:    "No discovery runs found",
			Subtitle: "Discovery runs will appear here",
		})
	}

	cards := make([]nodx.Node, 0, len(runs)+1)
	for _, run := range runs {
		cards = append(cards, discoveryRunCard(run))
	}

	if pagination.HasNextPage {
		cards = append(cards, nodx.Div(
			htmx.HxGet(buildListURL(q, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(cards)
}

func discoveryRunCard(run dbgen.DiscoveryServicePaginateRunsRow) nodx.Node {
	status := "running"
	switch {
	case run.Finished && run.ErrorsCount == 0:
		status = "success"
	case run.Finished && run.ErrorsCount > 0:
		status = "failed"
	case isDiscoveryRunStale(run):
		status = "failed"
	}

	return component.ItemCard(
		[]nodx.Node{nodx.Data("row", "1")},
		[]nodx.Node{
			discoveryRunActions(run),
			component.StatusBadge(status),
			nodx.SpanEl(
				nodx.Class("font-semibold flex-1 truncate"),
				component.SpanText("Discovery run — "+discoveryRunTime(run.StartedAt)),
			),
		},
		[]nodx.Node{
			component.Stat("Ports scanned", component.SpanText(int32Text(run.PortsCount))),
			component.Stat("Databases found", component.SpanText(int32Text(run.DatabasesCount))),
			component.Stat("Created", component.SpanText(int32Text(run.DatabasesCreatedCount))),
			component.Stat("Backups created", component.SpanText(int32Text(run.BackupsCreatedCount))),
			component.Stat("Skipped", component.SpanText(int32Text(run.SkippedExistingCount))),
			component.Stat("Errors", component.SpanText(int32Text(run.ErrorsCount))),
			component.Stat("Updated", component.SpanText(discoveryRunTime(run.UpdatedAt))),
		},
	)
}

func discoveryRunModals(run dbgen.DiscoveryServicePaginateRunsRow) (nodx.Node, nodx.Node) {
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

	return logModal.HTML, reportModal.HTML
}

func discoveryRunActions(run dbgen.DiscoveryServicePaginateRunsRow) nodx.Node {
	runID := run.RunID.String()
	logModalID := "discovery-run-log-" + runID
	reportModalID := "discovery-run-report-" + runID

	logOpenerAttr := component.Modal(component.ModalParams{ID: logModalID}).OpenerAttr
	reportOpenerAttr := component.Modal(component.ModalParams{ID: reportModalID}).OpenerAttr

	return nodx.Div(
		nodx.Class("flex gap-1"),
		nodx.Div(
			nodx.Class("tooltip"),
			nodx.Data("tip", "Log"),
			nodx.Button(
				logOpenerAttr,
				nodx.Class("btn btn-square btn-sm btn-ghost"),
				lucide.Eye(),
			),
		),
		nodx.Div(
			nodx.Class("tooltip"),
			nodx.Data("tip", "Report"),
			nodx.Button(
				reportOpenerAttr,
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
