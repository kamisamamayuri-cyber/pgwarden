package middleware

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/labstack/echo/v4"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

func (m *Middleware) RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()
		reqCtx := reqctx.GetCtx(c)

		if reqCtx.IsAuthed {
			return next(c)
		}

		usersQty, err := m.servs.UsersService.GetUsersQty(ctx)
		if err != nil {
			logger.Error("failed to get users qty", logger.KV{
				"ip":    c.RealIP(),
				"ua":    c.Request().UserAgent(),
				"error": err,
			})
			return c.String(http.StatusInternalServerError, i18n.ErrInternalServer)
		}

		if usersQty == 0 {
			redirectPath := pathutil.BuildPath("/auth/create-first-user")
			htmx.ServerSetRedirect(c.Response().Header(), redirectPath)
			return c.Redirect(http.StatusFound, redirectPath)
		}

		redirectPath := pathutil.BuildPath("/auth/login")
		htmx.ServerSetRedirect(c.Response().Header(), redirectPath)
		return c.Redirect(http.StatusFound, redirectPath)
	}
}
