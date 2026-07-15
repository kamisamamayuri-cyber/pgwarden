package middleware

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/labstack/echo/v4"
)

// RequireRestoreAccess allows only users who can view at least one restore preset.
func (m *Middleware) RequireRestoreAccess(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !m.servs.RbacService.Enabled() {
			return next(c)
		}
		rc := reqctx.GetCtx(c)
		access := m.servs.RbacService.Access(rc.Groups, rc.FullAccess)
		for _, p := range restorations.GetPresets() {
			if access.CanViewPreset(p.ID) {
				return next(c)
			}
		}
		return c.String(http.StatusForbidden, "Access denied")
	}
}
