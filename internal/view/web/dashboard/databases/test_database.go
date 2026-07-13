package databases

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *handlers) testDatabaseHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	var formData createDatabaseDTO
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	err := h.servs.DatabasesService.TestDatabase(
		ctx, formData.Version, formData.ConnectionString,
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.ToastSuccess(c, i18n.MsgConnectionOK)
}

func (h *handlers) testExistingDatabaseHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	databaseID, err := uuid.Parse(c.Param("databaseID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	err = h.servs.DatabasesService.TestDatabaseAndStoreResult(ctx, databaseID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.ToastSuccess(c, i18n.MsgConnectionOK)
}
