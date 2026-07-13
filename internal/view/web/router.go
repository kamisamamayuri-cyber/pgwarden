package web

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/auth"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/labstack/echo/v4"
)

func MountRouter(
	parent *echo.Group, mids *middleware.Middleware, servs *service.Service,
) {
	// GET / -> Handle the root path redirects
	parent.GET("", func(c echo.Context) error {
		ctx := c.Request().Context()
		reqCtx := reqctx.GetCtx(c)

		if reqCtx.IsAuthed {
			return c.Redirect(http.StatusFound, pathutil.BuildPath("/dashboard"))
		}

		usersQty, err := servs.UsersService.GetUsersQty(ctx)
		if err != nil {
			logger.Error("failed to get users qty", logger.KV{
				"ip":    c.RealIP(),
				"ua":    c.Request().UserAgent(),
				"error": err,
			})
			return c.String(http.StatusInternalServerError, i18n.ErrInternalServer)
		}

		if usersQty == 0 {
			return c.Redirect(http.StatusFound, pathutil.BuildPath("/auth/create-first-user"))
		}

		return c.Redirect(http.StatusFound, pathutil.BuildPath("/auth/login"))
	})

	authGroup := parent.Group("/auth")
	auth.MountRouter(authGroup, mids, servs)

	dashboardGroup := parent.Group("/dashboard", mids.RequireAuth)
	dashboard.MountRouter(dashboardGroup, mids, servs)
}
