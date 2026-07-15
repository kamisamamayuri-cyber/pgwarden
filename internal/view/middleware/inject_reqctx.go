package middleware

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/labstack/echo/v4"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

func (m *Middleware) InjectReqctx(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqCtx := reqctx.Ctx{
			IsHTMXBoosted: htmx.ServerGetIsBoosted(c.Request().Header),
		}

		found, user, err := m.servs.AuthService.GetUserFromSessionCookie(c)
		if err != nil {
			logger.Error("failed to get user from session cookie", logger.KV{
				"ip":    c.RealIP(),
				"ua":    c.Request().UserAgent(),
				"error": err,
			})
			return c.String(http.StatusInternalServerError, i18n.ErrInternalServer)
		}

		if found {
			reqCtx.IsAuthed = true
			reqCtx.FullAccess = user.FullAccess
			reqCtx.Groups = user.Groups
			reqCtx.Access = m.servs.RbacService.Access(user.Groups, user.FullAccess)
			reqCtx.SessionID = user.SessionID
			reqCtx.User = dbgen.User{
				ID:        user.ID,
				Name:      user.Name,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			}
		}

		reqctx.SetCtx(c, reqCtx)
		return next(c)
	}
}
