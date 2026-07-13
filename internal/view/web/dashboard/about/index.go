package about

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
)

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)
	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx))
}

func indexPage(reqCtx reqctx.Ctx) nodx.Node {
	content := []nodx.Node{
		component.H1Text("About"),

		nodx.Div(
			nodx.Class("grid grid-cols-1 desk:grid-cols-2 gap-4 mt-4"),

			component.CardBox(component.CardBoxParams{
				Children: []nodx.Node{
					component.PText(`
						PostgreSQL backup service. Configure schedules, store backups
						locally or in S3-compatible storage, and restore databases
						through the web interface.
					`),
				},
			}),

			component.CardBox(component.CardBoxParams{
				Children: []nodx.Node{
					nodx.Table(
						nodx.Class("table"),
						nodx.Tr(
							nodx.Th(component.SpanText("Product")),
							nodx.Td(component.SpanText(component.AppName)),
						),
						nodx.Tr(
							nodx.Th(component.SpanText("License")),
							nodx.Td(
								nodx.A(
									nodx.Class("link"),
									nodx.Href(component.RepoURL + "/-/blob/master/LICENSE"),
									nodx.Target("_blank"),
									component.SpanText("AGPL v3 (upstream PG Warden)"),
								),
							),
						),
						nodx.Tr(
							nodx.Th(component.SpanText("Repository")),
							nodx.Td(
								nodx.A(
									nodx.Class("link"),
									nodx.Href(component.RepoURL),
									nodx.Target("_blank"),
									component.SpanText("github.com/kamisamamayuri-cyber/pgwarden"),
								),
							),
						),
					),
				},
			}),
		),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "About",
		Body:  content,
	})
}
