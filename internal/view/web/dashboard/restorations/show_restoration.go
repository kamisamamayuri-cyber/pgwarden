package restorations

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func showRestorationButton(
	restoration dbgen.RestorationsServicePaginateRestorationsRow,
) nodx.Node {
	view := restorationDetailsViewFromPaginate(restoration)

	mo := component.Modal(component.ModalParams{
		ID:            restorationModalID(restoration.ID),
		Title:         "Restoration details",
		Size:          component.SizeMd,
		HTMXIndicator: restorationDetailsLoadingID(restoration.ID),
		Content: []nodx.Node{
			renderRestorationDetails(view, view.Status == "running"),
		},
	})

	button := nodx.Button(
		mo.OpenerAttr,
		nodx.Class("btn btn-square btn-sm btn-ghost"),
		lucide.Eye(),
	)

	return nodx.Div(
		nodx.Class("inline-block tooltip tooltip-right"),
		nodx.Data("tip", i18n.BtnShowDetails),
		mo.HTML,
		button,
	)
}
