package databases

import (
	"fmt"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) deleteDatabaseHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	databaseID, err := uuid.Parse(c.Param("databaseID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if err = h.servs.DatabasesService.DeleteDatabase(ctx, databaseID); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.Refresh(c)
}

func deleteDatabaseButton(databaseID uuid.UUID) nodx.Node {
	return component.OptionsDropdownButton(
		htmx.HxDelete(pathutil.BuildPath(fmt.Sprintf("/dashboard/databases/%s", databaseID))),
		htmx.HxConfirm("Delete this database?"),
		lucide.Trash(),
		component.SpanText("Delete database"),
	)
}
