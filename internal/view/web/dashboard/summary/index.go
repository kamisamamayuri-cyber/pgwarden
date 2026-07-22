package summary

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

const defaultDays = 30
const healthListLimit = 10

// healthEntry is a name+status pair used to sort/truncate both the
// databases and destinations health lists the same way.
type healthEntry struct {
	name string
	ok   sql.NullBool
}

func moreSummaryText(moreHealthy, moreProblems int) string {
	total := moreHealthy + moreProblems
	if moreProblems == 0 {
		return fmt.Sprintf("+ %d more, all healthy", total)
	}
	return fmt.Sprintf("+ %d more (%d with issues)", total, moreProblems)
}

func healthEntryIsProblem(ok sql.NullBool) bool {
	return !ok.Valid || !ok.Bool
}

// sortAndLimitHealth puts unhealthy/untested entries first (most actionable),
// then healthy ones, and returns at most healthListLimit rows plus how many
// healthy and how many problem entries were cut off.
func sortAndLimitHealth(entries []healthEntry) (shown []healthEntry, moreHealthy, moreProblems int) {
	sorted := make([]healthEntry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		return healthEntryIsProblem(sorted[i].ok) && !healthEntryIsProblem(sorted[j].ok)
	})
	if len(sorted) <= healthListLimit {
		return sorted, 0, 0
	}
	shown = sorted[:healthListLimit]
	for _, e := range sorted[healthListLimit:] {
		if healthEntryIsProblem(e.ok) {
			moreProblems++
		} else {
			moreHealthy++
		}
	}
	return shown, moreHealthy, moreProblems
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	ctx := c.Request().Context()
	reqCtx := reqctx.GetCtx(c)

	days := defaultDays
	if d := c.QueryParam("days"); d != "" {
		switch d {
		case "7":
			days = 7
		case "90":
			days = 90
		default:
			days = defaultDays
		}
	}

	execPerDay, err := h.servs.ExecutionsService.GetExecutionsPerDay(ctx, int32(days))
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	restPerDay, err := h.servs.RestorationsService.GetRestorationsPerDay(ctx, int32(days))
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	databases, err := h.servs.DatabasesService.GetDatabasesHealth(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	executionsQty, err := h.servs.ExecutionsService.GetExecutionsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	restorationsQty, err := h.servs.RestorationsService.GetRestorationsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	var destinations []dbgen.DestinationsServiceGetDestinationsHealthRow
	if reqCtx.Access.CanManageApp() {
		destinations, err = h.servs.DestinationsService.GetDestinationsHealth(ctx)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(
		reqCtx, days, execPerDay, restPerDay, databases, destinations,
		executionsQty, restorationsQty,
	))
}

func indexPage(
	reqCtx reqctx.Ctx,
	days int,
	execPerDay []dbgen.ExecutionsServiceGetExecutionsPerDayRow,
	restPerDay []dbgen.RestorationsServiceGetRestorationsPerDayRow,
	databases []dbgen.DatabasesServiceGetDatabasesHealthRow,
	destinations []dbgen.DestinationsServiceGetDestinationsHealthRow,
	executionsQty dbgen.ExecutionsServiceGetExecutionsQtyRow,
	restorationsQty dbgen.RestorationsServiceGetRestorationsQtyRow,
) nodx.Node {
	buildURL := func(d int) string {
		return pathutil.BuildPath(fmt.Sprintf("/dashboard?days=%d", d))
	}

	periodBtn := func(label string, d int) nodx.Node {
		active := days == d
		cls := "join-item btn btn-sm"
		if active {
			cls += " btn-primary"
		} else {
			cls += " btn-ghost"
		}
		return nodx.A(
			nodx.Class(cls),
			nodx.Href(buildURL(d)),
			htmx.HxBoost("true"),
			htmx.HxTarget("#dashboard-main"),
			htmx.HxSwap("transition:true show:unset"),
			nodx.Text(label),
		)
	}

	statCard := func(label string, value int64, sub string, subOk bool) nodx.Node {
		subCls := "text-xs mt-1"
		if subOk {
			subCls += " text-success"
		} else {
			subCls += " text-error"
		}
		return component.CardBox(component.CardBoxParams{
			Children: []nodx.Node{
				nodx.Div(
					nodx.Class("text-xs text-base-content/50 uppercase tracking-wide"),
					nodx.Text(label),
				),
				nodx.Div(
					nodx.Class("text-2xl font-bold mt-1"),
					nodx.Text(fmt.Sprintf("%d", value)),
				),
				nodx.Div(
					nodx.Class(subCls),
					nodx.Text(sub),
				),
			},
		})
	}

	buildChartScript := func(
		chartID string,
		labels []string,
		successData []int32,
		failedData []int32,
	) nodx.Node {
		ls := make([]string, len(labels))
		for i, l := range labels {
			ls[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(l, "'", "\\'"))
		}
		sd := make([]string, len(successData))
		for i, v := range successData {
			sd[i] = fmt.Sprintf("%d", v)
		}
		fd := make([]string, len(failedData))
		for i, v := range failedData {
			fd[i] = fmt.Sprintf("%d", v)
		}
		return nodx.Script(nodx.Raw(fmt.Sprintf(`
new Chart(document.getElementById('%s'), {
  type: 'bar',
  data: {
    labels: [%s],
    datasets: [
      { label: 'Success', data: [%s], backgroundColor: '#15803d99', stack: 's' },
      { label: 'Failed',  data: [%s], backgroundColor: '#FF001F99', stack: 's' }
    ]
  },
  options: {
    responsive: true,
    maintainAspectRatio: false,
    plugins: { legend: { position: 'bottom', labels: { color: 'rgba(255,255,255,0.5)', boxWidth: 12 } } },
    scales: {
      x: { stacked: true, ticks: { color: 'rgba(255,255,255,0.4)', font: { size: 10 } }, grid: { color: 'rgba(255,255,255,0.05)' } },
      y: { stacked: true, ticks: { color: 'rgba(255,255,255,0.4)', font: { size: 10 } }, grid: { color: 'rgba(255,255,255,0.05)' } }
    }
  }
});
`, chartID, strings.Join(ls, ","), strings.Join(sd, ","), strings.Join(fd, ","))))
	}

	execLabels := make([]string, len(execPerDay))
	execSuccess := make([]int32, len(execPerDay))
	execFailed := make([]int32, len(execPerDay))
	execTotal := int64(0)
	execFailTotal := int64(0)
	for i, row := range execPerDay {
		execLabels[i] = row.Day.Format("Jan 2")
		execSuccess[i] = row.Success
		execFailed[i] = row.Failed
		execTotal += int64(row.Success) + int64(row.Failed)
		execFailTotal += int64(row.Failed)
	}

	restLabels := make([]string, len(restPerDay))
	restSuccess := make([]int32, len(restPerDay))
	restFailed := make([]int32, len(restPerDay))
	restTotal := int64(0)
	restFailTotal := int64(0)
	for i, row := range restPerDay {
		restLabels[i] = row.Day.Format("Jan 2")
		restSuccess[i] = row.Success
		restFailed[i] = row.Failed
		restTotal += int64(row.Success) + int64(row.Failed)
		restFailTotal += int64(row.Failed)
	}

	// unique IDs: with hx-boost transition swaps the old canvas may still be
	// in the DOM when the new script runs, and Chart.js refuses to reuse one
	execChartID := "chart-exec-" + uuid.NewString()
	restChartID := "chart-rest-" + uuid.NewString()

	healthBadge := func(ok sql.NullBool, okLabel, failLabel string) nodx.Node {
		if !ok.Valid {
			return nodx.SpanEl(
				nodx.Class("badge badge-xs border-0 bg-base-content/10 text-base-content/50"),
				nodx.Text("untested"),
			)
		}
		if ok.Bool {
			return nodx.SpanEl(
				nodx.Class("badge badge-xs border-0 bg-success/20 text-success"),
				nodx.Text(okLabel),
			)
		}
		return nodx.SpanEl(
			nodx.Class("badge badge-xs border-0 bg-error/20 text-error"),
			nodx.Text(failLabel),
		)
	}

	dbEntries := make([]healthEntry, 0, len(databases))
	for _, db := range databases {
		dbEntries = append(dbEntries, healthEntry{name: db.Name, ok: db.TestOk})
	}
	dbShown, dbMoreHealthy, dbMoreProblems := sortAndLimitHealth(dbEntries)
	dbRows := make([]nodx.Node, 0, len(dbShown))
	for _, db := range dbShown {
		dbRows = append(dbRows, nodx.Tr(
			nodx.Td(nodx.Text(db.name)),
			nodx.Td(healthBadge(db.ok, "online", "offline")),
		))
	}

	dbHealthy := 0
	dbUnhealthy := 0
	for _, db := range databases {
		if !db.TestOk.Valid {
			continue
		}
		if db.TestOk.Bool {
			dbHealthy++
		} else {
			dbUnhealthy++
		}
	}

	dstEntries := make([]healthEntry, 0, len(destinations))
	for _, dst := range destinations {
		dstEntries = append(dstEntries, healthEntry{name: dst.Name, ok: dst.TestOk})
	}
	dstShown, dstMoreHealthy, dstMoreProblems := sortAndLimitHealth(dstEntries)
	dstRows := make([]nodx.Node, 0, len(dstShown))
	for _, dst := range dstShown {
		dstRows = append(dstRows, nodx.Tr(
			nodx.Td(nodx.Text(dst.name)),
			nodx.Td(healthBadge(dst.ok, "available", "unavailable")),
		))
	}

	dstHealthy := 0
	dstUnhealthy := 0
	for _, dst := range destinations {
		if !dst.TestOk.Valid {
			continue
		}
		if dst.TestOk.Bool {
			dstHealthy++
		} else {
			dstUnhealthy++
		}
	}

	healthSection := func(
		title string,
		total int,
		healthy, unhealthy int,
		icon func(...nodx.Node) nodx.Node,
		tableRows []nodx.Node,
		moreHealthy, moreProblems int,
	) nodx.Node {
		subtitle := fmt.Sprintf("%d total", total)
		return component.CardBox(component.CardBoxParams{
			Children: []nodx.Node{
				nodx.Div(
					nodx.Class("flex items-center gap-2 mb-3"),
					icon(nodx.Class("size-4 text-base-content/50")),
					nodx.SpanEl(
						nodx.Class("text-sm font-semibold text-base-content/70 uppercase tracking-wide"),
						nodx.Text(title),
					),
					nodx.SpanEl(
						nodx.Class("text-xs text-base-content/40"),
						nodx.Text(subtitle),
					),
					nodx.Div(nodx.Class("flex-1")),
					nodx.If(healthy > 0, nodx.SpanEl(
						nodx.Class("badge badge-xs border-0 bg-success/20 text-success"),
						nodx.Text(fmt.Sprintf("%d up", healthy)),
					)),
					nodx.If(unhealthy > 0, nodx.SpanEl(
						nodx.Class("badge badge-xs border-0 bg-error/20 text-error ml-1"),
						nodx.Text(fmt.Sprintf("%d down", unhealthy)),
					)),
				),
				nodx.If(len(tableRows) > 0,
					nodx.Table(
						nodx.Class("table table-xs"),
						nodx.Tbody(nodx.Group(tableRows...)),
					),
				),
				nodx.If(len(tableRows) == 0,
					nodx.Div(
						nodx.Class("text-sm text-base-content/40 py-2"),
						nodx.Text("No entries"),
					),
				),
				nodx.If(moreHealthy+moreProblems > 0,
					nodx.Div(
						nodx.Class("text-xs text-base-content/40 pt-1"),
						nodx.Text(moreSummaryText(moreHealthy, moreProblems)),
					),
				),
			},
		})
	}

	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex items-center justify-between gap-4 flex-wrap"),
			component.H1Text("Overview"),
			nodx.Div(
				nodx.Class("join"),
				periodBtn("7d", 7),
				periodBtn("30d", 30),
				periodBtn("90d", 90),
			),
		),

		nodx.Div(
			nodx.Class("mt-4 grid grid-cols-2 desk:grid-cols-4 gap-3"),
			statCard("Jobs", execTotal, fmt.Sprintf("%d total ever", executionsQty.All), true),
			statCard("Job failures", execFailTotal, fmt.Sprintf("period: last %dd", days), execFailTotal == 0),
			statCard("Restorations", restTotal, fmt.Sprintf("%d total ever", restorationsQty.All), true),
			statCard("Rest. failures", restFailTotal, fmt.Sprintf("period: last %dd", days), restFailTotal == 0),
		),

		nodx.Div(
			nodx.Class("mt-4 grid grid-cols-1 desk:grid-cols-2 gap-4"),
			component.CardBox(component.CardBoxParams{
				Children: []nodx.Node{
					nodx.Div(
						nodx.Class("flex items-center gap-2 mb-3"),
						lucide.DatabaseBackup(nodx.Class("size-4 text-base-content/50")),
						nodx.SpanEl(
							nodx.Class("text-sm font-semibold text-base-content/70 uppercase tracking-wide"),
							nodx.Text("Backup Jobs / day"),
						),
					),
					nodx.Div(
						nodx.Class("relative h-64"),
						nodx.Canvas(nodx.Id(execChartID)),
					),
					buildChartScript(execChartID, execLabels, execSuccess, execFailed),
				},
			}),
			component.CardBox(component.CardBoxParams{
				Children: []nodx.Node{
					nodx.Div(
						nodx.Class("flex items-center gap-2 mb-3"),
						lucide.ArchiveRestore(nodx.Class("size-4 text-base-content/50")),
						nodx.SpanEl(
							nodx.Class("text-sm font-semibold text-base-content/70 uppercase tracking-wide"),
							nodx.Text("Restorations / day"),
						),
					),
					nodx.Div(
						nodx.Class("relative h-64"),
						nodx.Canvas(nodx.Id(restChartID)),
					),
					buildChartScript(restChartID, restLabels, restSuccess, restFailed),
				},
			}),
		),

		nodx.Div(
			nodx.Class("mt-4 grid grid-cols-1 desk:grid-cols-2 gap-4"),
			healthSection(
				"Databases", len(databases), dbHealthy, dbUnhealthy,
				lucide.Database, dbRows, dbMoreHealthy, dbMoreProblems,
			),
			nodx.If(reqCtx.Access.CanManageApp(),
				healthSection(
					"Destinations", len(destinations), dstHealthy, dstUnhealthy,
					lucide.HardDrive, dstRows, dstMoreHealthy, dstMoreProblems,
				),
			),
		),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Overview",
		Body:  content,
	})
}
