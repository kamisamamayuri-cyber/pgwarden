package layout

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func dashboardHeader() nodx.Node {
	return nodx.Header(
		nodx.ClassMap{
			"sticky top-0 z-50":       true,
			"space-x-4 p-4 min-w-max": true,
			"w-[full] bg-base-100 border-b border-base-300 shadow-sm": true,
			"flex items-center justify-between":                       true,
		},
		nodx.Div(
			nodx.Class("flex justify-start items-center space-x-2"),
			component.ChangeThemeButton(component.ChangeThemeButtonParams{
				Position: component.DropdownPositionBottom,
				Size:     component.SizeSm,
			}),
		),
		nodx.Div(
			nodx.Class("flex justify-end items-center space-x-2"),
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath("/dashboard/version-button")),
				htmx.HxSwap("outerHTML"),
				htmx.HxTrigger("load once"),
			),
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath("/dashboard/health-button")),
				htmx.HxSwap("outerHTML"),
				htmx.HxTrigger("load once"),
			),
			nodx.Button(
				htmx.HxPost(pathutil.BuildPath("/auth/logout")),
				htmx.HxDisabledELT("this"),
				nodx.Class("btn btn-ghost btn-neutral"),
				component.SpanText("Logout"),
				lucide.LogOut(),
			),
		),
	)
}
