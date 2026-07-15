package middleware

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/labstack/echo/v4"
)

func (m *Middleware) RequireManageApp(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		rc := reqctx.GetCtx(c)
		access := m.servs.RbacService.Access(rc.Groups, rc.FullAccess)
		if !access.CanManageApp() {
			return c.String(http.StatusForbidden, "Access denied")
		}
		return next(c)
	}
}
