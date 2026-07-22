package executions

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) deleteExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	err = h.servs.ExecutionsService.SoftDeleteExecution(ctx, executionID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.Refresh(c)
}

func deleteExecutionButton(executionID uuid.UUID) nodx.Node {
	return nodx.Button(
		htmx.HxDelete(pathutil.BuildPath(fmt.Sprintf("/dashboard/jobs/%s", executionID))),
		htmx.HxDisabledELT("this"),
		htmx.HxConfirm("Delete this execution? The backup file will be permanently removed from storage."),
		nodx.Class("btn btn-error btn-outline"),
		component.SpanText(i18n.BtnDelete),
		lucide.Trash(),
	)
}
