package executions

import (
	"net/http"

	"github.com/google/uuid"
	restorationsService "github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type execsQueryData struct {
	Database    uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup      uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	Status      string    `query:"status" validate:"omitempty,oneof=queued running success failed deleted"`
	Type        string    `query:"type" validate:"omitempty,oneof=backup restore fix_owner"`
	Host        string    `query:"host" validate:"omitempty,max=253"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData execsQueryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func hasNewJobWizardAccess(reqCtx reqctx.Ctx) bool {
	for _, p := range restorationsService.GetPresets() {
		if reqCtx.Access.CanViewPreset(p.ID) {
			return true
		}
	}
	return false
}

func indexPage(reqCtx reqctx.Ctx, queryData execsQueryData) nodx.Node {
	filterQuery := execsFilterQuery(queryData)

	var headerRight nodx.Node
	var wizardModal nodx.Node
	if hasNewJobWizardAccess(reqCtx) {
		headerRight = restorations.WizardOpenButton("New Job", "/dashboard/restorations/wizard/step0")
		wizardModal = restorations.WizardModal("New Job")
	}

	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex items-start justify-between gap-4"),
			component.H1Text("Jobs"),
			headerRight,
		),
		wizardModal,
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				executionHostFilter(queryData.Host, filterQuery),
				executionStatusFilters(filterQuery),
				executionTypeFilters(filterQuery),
				nodx.Div(
					nodx.Id(executionsTbodyID),
					component.SkeletonCard(6),
					htmx.HxGet(buildExecutionsListURL(filterQuery, 1)),
					htmx.HxTrigger("load"),
					htmx.HxSwap("outerHTML"),
				),
			},
		}),
		nodx.Div(nodx.Id(executionModalsContainerID)),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Jobs",
		Body:  content,
	})
}

func executionHostFilter(host string, q execsFilterQuery) nodx.Node {
	resetQuery := q
	resetQuery.Host = ""
	return nodx.FormEl(
		nodx.Class("mb-4 flex flex-wrap items-end gap-2"),
		nodx.Method("GET"),
		nodx.Action(buildExecutionsIndexURL(resetQuery, q.Status, q.Type)),
		nodx.Div(
			nodx.Class("form-control w-full max-w-xs"),
			nodx.LabelEl(
				nodx.Class("label py-0"),
				nodx.For("executions-host-filter"),
				component.SpanText(i18n.LabelHost),
			),
			nodx.Input(
				nodx.Class("input input-bordered input-sm w-full"),
				nodx.Id("executions-host-filter"),
				nodx.Name("host"),
				nodx.Type("text"),
				nodx.Placeholder("db-prod-01"),
				nodx.Value(host),
			),
		),
		nodx.Button(
			nodx.Class("btn btn-sm btn-primary"),
			nodx.Type("submit"),
			nodx.Text("Apply"),
		),
		nodx.If(
			host != "",
			nodx.A(
				nodx.Class("btn btn-sm btn-ghost"),
				nodx.Href(buildExecutionsIndexURL(resetQuery, q.Status, q.Type)),
				nodx.Text("Reset"),
			),
		),
	)
}

func executionStatusFilters(q execsFilterQuery) nodx.Node {
	filters := []struct {
		label  string
		status string
	}{
		{label: "All", status: ""},
		{label: i18n.StatusLabel("queued"), status: "queued"},
		{label: i18n.StatusLabel("running"), status: "running"},
		{label: i18n.StatusLabel("success"), status: "success"},
		{label: i18n.StatusLabel("failed"), status: "failed"},
		{label: i18n.StatusLabel("deleted"), status: "deleted"},
	}

	buttons := make([]nodx.Node, 0, len(filters))
	for _, filter := range filters {
		active := q.Status == filter.status
		btnClass := "btn btn-sm"
		if active {
			btnClass += " btn-primary"
		} else {
			btnClass += " btn-ghost"
		}

		buttons = append(buttons, nodx.A(
			nodx.Class(btnClass),
			nodx.Href(buildExecutionsIndexURL(q, filter.status, q.Type)),
			nodx.Text(filter.label),
		))
	}

	return nodx.Div(
		nodx.Class("mb-4 flex flex-wrap gap-2"),
		nodx.SpanEl(
			nodx.Class("text-sm text-base-content/70 self-center mr-1"),
			nodx.Text(i18n.LabelStatus+":"),
		),
		component.RenderableGroup(buttons),
	)
}

func executionTypeFilters(q execsFilterQuery) nodx.Node {
	filters := []struct {
		label string
		typ   string
	}{
		{label: "All", typ: ""},
		{label: "Backup", typ: "backup"},
		{label: "Restore", typ: "restore"},
		{label: "Fix owner", typ: "fix_owner"},
	}

	buttons := make([]nodx.Node, 0, len(filters))
	for _, filter := range filters {
		active := q.Type == filter.typ
		btnClass := "btn btn-sm"
		if active {
			btnClass += " btn-primary"
		} else {
			btnClass += " btn-ghost"
		}

		buttons = append(buttons, nodx.A(
			nodx.Class(btnClass),
			nodx.Href(buildExecutionsIndexURL(q, q.Status, filter.typ)),
			nodx.Text(filter.label),
		))
	}

	return nodx.Div(
		nodx.Class("mb-4 flex flex-wrap gap-2"),
		nodx.SpanEl(
			nodx.Class("text-sm text-base-content/70 self-center mr-1"),
			nodx.Text("Type:"),
		),
		component.RenderableGroup(buttons),
	)
}
