package restorations

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func restorationModalTemplate(
	restoration dbgen.RestorationsServicePaginateRestorationsRow,
) nodx.Node {
	view := restorationDetailsViewFromPaginate(restoration)

	mo := component.Modal(component.ModalParams{
		ID:            restorationModalID(restoration.ID),
		Title:         "Restoration details",
		Size:          component.SizeMd,
		HTMXIndicator: restorationDetailsLoadingID(restoration.ID),
		Content: []nodx.Node{
			renderRestorationDetails(view, view.Status == "running" || view.Status == "queued"),
		},
	})
	return mo.HTML
}

func showRestorationDropdownItem(
	restoration dbgen.RestorationsServicePaginateRestorationsRow,
) nodx.Node {
	opener := component.Modal(component.ModalParams{
		ID: restorationModalID(restoration.ID),
	}).OpenerAttr

	return component.OptionsDropdownButton(
		opener,
		lucide.Eye(),
		component.SpanText(i18n.BtnShowDetails),
	)
}
