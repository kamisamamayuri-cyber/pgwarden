package restorations

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

const WizardModalID = wizardModalID

func ModalTemplate(row dbgen.RestorationsServicePaginateRestorationsRow) nodx.Node {
	return restorationModalTemplate(row)
}

func ShowDetailsMenuItem(row dbgen.RestorationsServicePaginateRestorationsRow) nodx.Node {
	return showRestorationDropdownItem(row)
}

func TargetDatabase(row dbgen.RestorationsServicePaginateRestorationsRow) string {
	return restorationTargetDatabase(row)
}

func WizardModal(title string) nodx.Node {
	return component.Modal(component.ModalParams{
		ID:    wizardModalID,
		Size:  component.SizeLg,
		Title: title,
		Content: []nodx.Node{
			nodx.Div(nodx.Id("wizard-slot"), nodx.Class("min-h-[120px]")),
		},
	}).HTML
}

func WizardOpenButton(label, startPath string) nodx.Node {
	return nodx.Button(
		nodx.Class("btn btn-primary"),
		component.SpanText(label),
		lucide.Plus(),
		htmx.HxGet(pathutil.BuildPath(startPath)),
		htmx.HxTarget("#wizard-slot"),
		htmx.HxSwap("innerHTML"),
		nodx.Attr("onClick", fmt.Sprintf("window.dispatchEvent(new Event('%s_open'));", wizardModalID)),
	)
}
