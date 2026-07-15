package layout

import (
	"fmt"

	nodx "github.com/nodxdev/nodxgo"
	alpine "github.com/nodxdev/nodxgo-alpine"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
)

func dashboardAside(reqCtx reqctx.Ctx) nodx.Node {
	access := reqCtx.Access

	items := []nodx.Node{
		dashboardAsideItem(
			lucide.LayoutDashboard,
			"Overview",
			pathutil.BuildPath("/dashboard"),
			true,
		),
	}

	if access.CanManageApp() {
		items = append(items,
			dashboardAsideItem(
				lucide.Search,
				"Discovery",
				pathutil.BuildPath("/dashboard/discovery"),
				false,
			),
			dashboardAsideItem(
				lucide.FileCode2,
				"Configs",
				pathutil.BuildPath("/dashboard/configs"),
				false,
			),
			dashboardAsideItem(
				lucide.HardDrive,
				"Destinations",
				pathutil.BuildPath("/dashboard/destinations"),
				false,
			),
		)
	}

	items = append(items,
		dashboardAsideItem(
			lucide.Database,
			"Databases",
			pathutil.BuildPath("/dashboard/databases"),
			false,
		),
		dashboardAsideItem(
			lucide.DatabaseBackup,
			"Backups",
			pathutil.BuildPath("/dashboard/backups"),
			false,
		),
		dashboardAsideItem(
			lucide.List,
			"Executions",
			pathutil.BuildPath("/dashboard/executions"),
			false,
		),
		dashboardAsideItem(
			lucide.ArchiveRestore,
			"Restorations",
			pathutil.BuildPath("/dashboard/restorations"),
			false,
		),
	)

	if access.CanManageApp() {
		items = append(items, dashboardAsideItem(
			lucide.ScrollText,
			"Logs",
			pathutil.BuildPath("/dashboard/logs"),
			false,
		))
	}

	items = append(items,
		dashboardAsideItem(
			lucide.User,
			"Profile",
			pathutil.BuildPath("/dashboard/profile"),
			false,
		),
		dashboardAsideItem(
			lucide.Info,
			"About",
			pathutil.BuildPath("/dashboard/about"),
			false,
		),
	)

	return nodx.Aside(
		nodx.Id("dashboard-aside"),
		nodx.ClassMap{
			"flex-none h-[100dvh] bg-base-100 border-r border-base-300 shadow-sm p-4": true,
			"overflow-y-auto overflow-x-hidden":                                       true,
		},

		nodx.A(
			nodx.Class("block"),
			nodx.Href(pathutil.BuildPath("/dashboard")),
			htmx.HxBoost("true"),
			htmx.HxTarget("#dashboard-main"),
			htmx.HxSwap("transition:true show:unset"),
			nodx.Div(
				nodx.Class("flex flex-col items-center justify-center"),
				component.Logotype(component.LogotypeParams{
					Compact: true,
					Size:    component.SizeSm,
				}),
			),
		),

		nodx.Div(
			nodx.Class("mt-6 space-y-4"),
			nodx.Group(items...),
		),
	)
}

func dashboardAsideItem(
	icon func(children ...nodx.Node) nodx.Node,
	text, link string, strict bool,
) nodx.Node {
	return nodx.A(
		alpine.XData(fmt.Sprintf("alpineDashboardAsideItem('%s', %t)", link, strict)),
		nodx.Class("block flex flex-col items-center justify-center group"),

		nodx.Href(link),
		htmx.HxBoost("true"),
		htmx.HxTarget("#dashboard-main"),
		htmx.HxSwap("transition:true show:unset"),

		nodx.Button(
			alpine.XBind("class", `{'btn-active btn-primary': is_active}`),
			nodx.Class("btn btn-ghost btn-neutral btn-square group-hover:btn-primary"),
			icon(nodx.Class("size-6")),
		),
		nodx.SpanEl(
			nodx.Class("text-xs text-center"),
			nodx.Text(text),
		),
	)
}
