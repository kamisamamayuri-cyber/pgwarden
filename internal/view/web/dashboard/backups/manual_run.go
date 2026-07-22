package backups

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auditlogs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) manualRunHandler(c echo.Context) error {
	ctx := c.Request().Context()

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, backupID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanExecutePbwName(backup.Name) {
		return webaccess.ForbiddenHTMX(c)
	}

	if err := h.servs.ExecutionsService.EnqueueExecution(ctx, backupID); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	h.servs.AuditLogsService.Log(ctx, auditlogs.Entry{
		UserEmail:   reqctx.GetCtx(c).User.Email,
		Action:      "run_backup",
		PresetID:    backup.ID.String(),
		PresetTitle: backup.Name,
		Source:      "ui",
	})

	return respondhtmx.ToastSuccess(c, "Backup queued, see details on the executions page")
}

func manualRunbutton(backupID uuid.UUID) nodx.Node {
	return component.OptionsDropdownButton(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/backups/%s/run", backupID))),
		htmx.HxDisabledELT("this"),
		lucide.Zap(),
		component.SpanText("Run now"),
	)
}
