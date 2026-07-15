package databases

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type databasesQueryData struct {
	Host string `query:"host" validate:"omitempty,max=253"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData databasesQueryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func indexPage(reqCtx reqctx.Ctx, queryData databasesQueryData) nodx.Node {
	filterQuery := databasesFilterQuery(queryData)

	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex justify-between items-start"),
			component.H1Text("Databases"),
			nodx.If(
				reqCtx.Access.CanManageApp(),
				createDatabaseButton(),
			),
		),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				databaseHostFilter(queryData.Host),
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText(i18n.LabelName)),
								nodx.Th(component.SpanText(i18n.LabelVersion)),
								nodx.Th(component.SpanText(i18n.LabelConnectionString)),
								nodx.Th(component.SpanText(i18n.LabelCreatedAt)),
							),
						),
						nodx.Tbody(
							component.SkeletonTr(8),
							htmx.HxGet(buildDatabasesListURL(filterQuery, 1)),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Databases",
		Body:  content,
	})
}

func databaseHostFilter(host string) nodx.Node {
	return nodx.FormEl(
		nodx.Class("mb-4 flex flex-wrap items-end gap-2"),
		nodx.Method("GET"),
		nodx.Action(buildDatabasesIndexURL("")),
		nodx.Div(
			nodx.Class("form-control w-full max-w-xs"),
			nodx.LabelEl(
				nodx.Class("label py-0"),
				nodx.For("databases-host-filter"),
				component.SpanText(i18n.LabelHost),
			),
			nodx.Input(
				nodx.Class("input input-bordered input-sm w-full"),
				nodx.Id("databases-host-filter"),
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
				nodx.Href(buildDatabasesIndexURL("")),
				nodx.Text("Reset"),
			),
		),
	)
}
