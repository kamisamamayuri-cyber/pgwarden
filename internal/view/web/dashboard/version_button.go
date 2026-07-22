package dashboard

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/versioncheck"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func versionButtonHandler(servs *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		return echoutil.RenderNodx(c, http.StatusOK, versionButton(servs.VersionCheckService))
	}
}

func versionButton(vc *versioncheck.Service) nodx.Node {
	if !vc.HasUpdate() {
		return nil
	}

	current := vc.CurrentVersion()
	latest := vc.LatestVersion()

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeSm,
		Title: "New version available",
		Content: []nodx.Node{
			nodx.Div(
				nodx.Class("space-y-3"),
				component.PText("A newer release of "+component.AppName+" is available on GitHub."),
				nodx.Div(
					nodx.Class("flex items-center gap-2 text-sm"),
					nodx.SpanEl(
						nodx.Class("text-base-content/60"),
						nodx.Text("Running"),
					),
					nodx.SpanEl(
						nodx.Class("badge badge-ghost font-mono"),
						nodx.Text(current),
					),
					nodx.SpanEl(
						nodx.Class("text-base-content/60"),
						nodx.Text("latest is"),
					),
					nodx.SpanEl(
						nodx.Class("badge badge-info text-white font-mono"),
						nodx.Text(latest),
					),
				),
				nodx.A(
					nodx.Class("btn btn-primary btn-sm w-full"),
					nodx.Href(component.RepoURL+"/releases/tag/"+latest),
					nodx.Target("_blank"),
					component.SpanText("View release on GitHub"),
					lucide.ExternalLink(nodx.Class("size-3.5")),
				),
			),
		},
	})

	return nodx.Div(
		nodx.Class("inline-block"),
		mo.HTML,
		nodx.Button(
			mo.OpenerAttr,
			nodx.Class("btn btn-ghost btn-neutral"),
			component.SpanText("Update available"),
			component.Ping(component.ColorInfo),
		),
	)
}
