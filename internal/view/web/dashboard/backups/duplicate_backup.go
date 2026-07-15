package backups

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

func (h *handlers) duplicateBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if _, err = h.servs.BackupsService.DuplicateBackup(ctx, backupID); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.Refresh(c)
}

func duplicateBackupButton(backupID uuid.UUID) nodx.Node {
	return component.OptionsDropdownButton(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/backups/%s/duplicate", backupID))),
		htmx.HxConfirm("Duplicate this backup?"),
		lucide.CopyPlus(),
		component.SpanText("Duplicate backup"),
	)
}
