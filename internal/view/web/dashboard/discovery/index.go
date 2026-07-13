package discovery

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
)

type queryData struct {
	Level    string `query:"level" validate:"omitempty,oneof=info error"`
	Event    string `query:"event" validate:"omitempty,max=64"`
	Host     string `query:"host" validate:"omitempty,max=253"`
	Port     int    `query:"port" validate:"omitempty,min=1,max=65535"`
	Database string `query:"database" validate:"omitempty,max=128"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData queryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func indexPage(reqCtx reqctx.Ctx, queryData queryData) nodx.Node {
	q := filterQuery(queryData)

	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex justify-between items-start gap-4"),
			nodx.Div(
				component.H1Text("Discovery"),
				component.PText("Discover PostgreSQL databases on specified hosts and view the registration log."),
			),
			nodx.Div(
				nodx.Class("flex gap-2"),
				nodx.Button(
					nodx.Class("btn btn-primary"),
					htmx.HxPost(buildRunURL(filterQuery{})),
					htmx.HxTarget("#discovery-events"),
					htmx.HxSwap("innerHTML"),
					lucide.Play(nodx.Class("size-4")),
					nodx.Text("Run all"),
				),
				nodx.Button(
					nodx.Class("btn btn-ghost"),
					htmx.HxGet(buildListURL(q, 1)),
					htmx.HxTarget("#discovery-events"),
					htmx.HxSwap("innerHTML"),
					lucide.RefreshCw(nodx.Class("size-4")),
					nodx.Text("Refresh log"),
				),
			),
		),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				discoveryFilter(q),
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText("Start")),
								nodx.Th(component.SpanText("Status")),
								nodx.Th(component.SpanText("Ports")),
								nodx.Th(component.SpanText("Databases")),
								nodx.Th(component.SpanText("New DBs")),
								nodx.Th(component.SpanText("New backups")),
								nodx.Th(component.SpanText("Already existed")),
								nodx.Th(component.SpanText("Errors")),
								nodx.Th(component.SpanText("Updated")),
							),
						),
						nodx.Tbody(
							nodx.Id("discovery-events"),
							component.SkeletonTr(7),
							htmx.HxGet(buildListURL(q, 1)),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Discovery",
		Body:  content,
	})
}

func discoveryFilter(q filterQuery) nodx.Node {
	return nodx.FormEl(
		nodx.Class("mb-4 grid grid-cols-1 md:grid-cols-6 gap-2 items-end"),
		nodx.Method("GET"),
		nodx.Action(buildIndexURL(filterQuery{})),
		filterInput("host", "Host", "db-prod-01", q.Host),
		filterInput("port", "Port", "16301", portValue(q.Port)),
		filterInput("database", "Database", "myapp", q.Database),
		filterSelect("level", "Level", q.Level, []filterOption{
			{Label: "All", Value: ""},
			{Label: "Info", Value: "info"},
			{Label: "Error", Value: "error"},
		}),
		filterInput("event", "Event", "backup_created", q.Event),
		nodx.Div(
			nodx.Class("flex gap-2"),
			nodx.Button(
				nodx.Class("btn btn-sm btn-primary"),
				nodx.Type("submit"),
				nodx.Text("Apply"),
			),
			nodx.A(
				nodx.Class("btn btn-sm btn-ghost"),
				nodx.Href(buildIndexURL(filterQuery{})),
				nodx.Text("Reset"),
			),
		),
	)
}

func filterInput(name, label, placeholder, value string) nodx.Node {
	id := "discovery-filter-" + name
	return nodx.Div(
		nodx.Class("form-control"),
		nodx.LabelEl(
			nodx.Class("label py-0"),
			nodx.For(id),
			component.SpanText(label),
		),
		nodx.Input(
			nodx.Class("input input-bordered input-sm w-full"),
			nodx.Id(id),
			nodx.Name(name),
			nodx.Type("text"),
			nodx.Placeholder(placeholder),
			nodx.Value(value),
		),
	)
}

type filterOption struct {
	Label string
	Value string
}

func filterSelect(name, label, value string, options []filterOption) nodx.Node {
	id := "discovery-filter-" + name
	optionNodes := make([]nodx.Node, 0, len(options))
	for _, option := range options {
		optionNodes = append(optionNodes, nodx.Option(
			nodx.Value(option.Value),
			nodx.If(option.Value == value, nodx.Selected("")),
			nodx.Text(option.Label),
		))
	}

	return nodx.Div(
		nodx.Class("form-control"),
		nodx.LabelEl(
			nodx.Class("label py-0"),
			nodx.For(id),
			component.SpanText(label),
		),
		nodx.Select(
			nodx.Class("select select-bordered select-sm w-full"),
			nodx.Id(id),
			nodx.Name(name),
			nodx.Group(optionNodes...),
		),
	)
}

func portValue(port int) string {
	if port == 0 {
		return ""
	}
	return strconv.Itoa(port)
}
