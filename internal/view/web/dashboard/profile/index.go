package profile

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
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
		component.H1Text("Profile"),
		nodx.Div(
			nodx.Class("mt-4 max-w-xl"),
			userInfoCard(reqCtx),
		),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Profile",
		Body:  content,
	})
}

func userInfoCard(reqCtx reqctx.Ctx) nodx.Node {
	user := reqCtx.User

	return component.CardBox(component.CardBoxParams{
		Children: []nodx.Node{
			component.H2Text("User info"),
			component.PText(`
				The account is managed via SSO. To update your information, make changes
				in the corporate directory.
			`),
			nodx.Div(
				nodx.Class("overflow-x-auto mt-4"),
				nodx.Table(
					nodx.Class("table"),
					nodx.Tr(
						nodx.Th(component.SpanText("Full name")),
						nodx.Td(component.SpanText(user.Name)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelEmail)),
						nodx.Td(component.SpanText(user.Email)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelCreatedAt)),
						nodx.Td(component.SpanText(
							user.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
						)),
					),
				),
			),
		},
	})
}
