package logs

import (
	"net/http"
	"strconv"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auditlogs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)
	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx))
}

func (h *handlers) listLogsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	page := 1
	if raw := c.QueryParam("page"); raw != "" {
		if p, err := strconv.Atoi(raw); err == nil && p > 0 {
			page = p
		}
	}

	rows, err := h.servs.AuditLogsService.List(ctx, page, 50)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, logRows(rows, page))
}

func indexPage(reqCtx reqctx.Ctx) nodx.Node {
	listURL := pathutil.BuildPath("/dashboard/logs/list")
	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex justify-between items-start gap-4"),
			nodx.Div(
				component.H1Text("Logs"),
				component.PText("Audit log of user actions: dump downloads and restore launches."),
			),
		),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(component.SpanText("Time")),
								nodx.Th(component.SpanText("User")),
								nodx.Th(component.SpanText("Action")),
								nodx.Th(component.SpanText("Preset")),
								nodx.Th(component.SpanText("Environment")),
								nodx.Th(component.SpanText("Source")),
							),
						),
						nodx.Tbody(
							nodx.Id("audit-log-rows"),
							component.SkeletonTr(6),
							htmx.HxGet(listURL),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Logs",
		Body:  content,
	})
}

func logRows(rows []auditlogs.LogRow, page int) nodx.Node {
	if len(rows) == 0 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No audit events yet",
			Subtitle: "Download and restore actions will appear here",
		})
	}

	nextURL := pathutil.BuildPath("/dashboard/logs/list") + "?page=" + strconv.Itoa(page+1)

	trs := make([]nodx.Node, 0, len(rows)+1)
	for _, r := range rows {
		trs = append(trs, logRow(r))
	}

	if len(rows) == 50 {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(nextURL),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}

func logRow(r auditlogs.LogRow) nodx.Node {
	env := ""
	if r.Environment != nil {
		env = *r.Environment
	}

	actionLabel := r.Action
	switch r.Action {
	case "download_dump":
		actionLabel = "Download dump"
	case "run_restore":
		actionLabel = "Run restore"
	}

	return nodx.Tr(
		nodx.Td(component.SpanText(r.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty))),
		nodx.Td(component.SpanText(r.UserEmail)),
		nodx.Td(component.SpanText(actionLabel)),
		nodx.Td(component.SpanText(r.PresetTitle)),
		nodx.Td(component.SpanText(env)),
		nodx.Td(component.SpanText(r.Source)),
	)
}
