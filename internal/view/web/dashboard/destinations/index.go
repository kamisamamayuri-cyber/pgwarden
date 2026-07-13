package destinations

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)
	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx))
}

func indexPage(reqCtx reqctx.Ctx) nodx.Node {
	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex justify-between items-start space-x-2"),
			nodx.Div(
				component.H1Text("Destinations"),
				component.PText(`
					Manage S3-compatible storage destinations. A destination is not required
					if backups are stored locally.
				`),
			),
			nodx.Div(
				nodx.Class("flex-none"),
				createDestinationButton(),
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
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText(i18n.LabelName)),
								nodx.Th(component.SpanText("Bucket name")),
								nodx.Th(component.SpanText(i18n.LabelEndpoint)),
								nodx.Th(component.SpanText("Region")),
								nodx.Th(component.SpanText("Access key")),
								nodx.Th(component.SpanText("Secret key")),
								nodx.Th(component.SpanText(i18n.LabelCreatedAt)),
							),
						),
						nodx.Tbody(
							component.SkeletonTr(8),
							htmx.HxGet(pathutil.BuildPath("/dashboard/destinations/list?page=1")),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Destinations",
		Body:  content,
	})
}
