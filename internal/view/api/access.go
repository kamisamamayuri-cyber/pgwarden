package api

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/labstack/echo/v4"
)

func (h *handlers) accessFrom(c echo.Context) rbac.Access {
	return reqctx.GetCtx(c).Access
}
