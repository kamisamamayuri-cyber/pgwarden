package layout

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	nodx "github.com/nodxdev/nodxgo"
)

type AuthParams struct {
	Title string
	Body  []nodx.Node
}

func Auth(params AuthParams) nodx.Node {
	title := component.AppName
	if params.Title != "" {
		title = params.Title + " — " + component.AppName
	}

	body := nodx.Group(
		nodx.ClassMap{
			"w-screen min-h-screen px-4 py-[40px]": true,
			"grid grid-cols-1 place-items-center": true,
			"brand-auth-shell overflow-y-auto":      true,
		},
		nodx.Div(
			nodx.Class("w-full max-w-[600px] space-y-4"),
			nodx.Div(
				nodx.Class("flex justify-center"),
				component.Logotype(component.LogotypeParams{Size: component.SizeLg}),
			),
			nodx.Main(
				nodx.Class("rounded-box shadow-lg bg-base-100 p-4 md:p-6"),
				nodx.Group(params.Body...),
			),
			nodx.Div(
				nodx.Class("flex justify-start"),
				component.ChangeThemeButton(component.ChangeThemeButtonParams{
					Position:    component.DropdownPositionTop,
					AlignsToEnd: false,
					Size:        component.SizeMd,
				}),
			),
		),
	)

	return commonHtmlDoc(title, body)
}
