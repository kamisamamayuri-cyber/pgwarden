package middleware

import (
	"net/http"
	"strings"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/labstack/echo/v4"
)

func (m *Middleware) RequireAuthAPI(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		if authHeader := strings.TrimSpace(c.Request().Header.Get("Authorization")); authHeader != "" {
			found, claims, err := m.servs.AuthService.GetUserFromBearerToken(ctx, authHeader)
			if err != nil {
				logger.Error("failed to validate bearer token", logger.KV{
					"ip":    c.RealIP(),
					"ua":    c.Request().UserAgent(),
					"error": err,
				})
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid bearer token",
				})
			}
			if found {
				name := strings.TrimSpace(claims.Name)
				if name == "" {
					name = strings.TrimSpace(claims.PreferredUser)
				}
				if name == "" {
					name = strings.ToLower(strings.TrimSpace(claims.Email))
				}

				reqctx.SetCtx(c, reqctx.Ctx{
					IsAuthed: true,
					Groups:   claims.Groups,
					Access:   m.servs.RbacService.Access(claims.Groups, false),
					User: dbgen.User{
						Name:  name,
						Email: strings.ToLower(strings.TrimSpace(claims.Email)),
					},
				})
				return next(c)
			}
		}

		found, user, err := m.servs.AuthService.GetUserFromSessionCookie(c)
		if err != nil {
			logger.Error("failed to get user from session cookie", logger.KV{
				"ip":    c.RealIP(),
				"ua":    c.Request().UserAgent(),
				"error": err,
			})
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "internal server error",
			})
		}

		if !found {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "unauthorized",
			})
		}

		reqctx.SetCtx(c, reqctx.Ctx{
			IsAuthed:   true,
			FullAccess: user.FullAccess,
			Groups:     user.Groups,
			Access:     m.servs.RbacService.Access(user.Groups, user.FullAccess),
			SessionID:  user.SessionID,
			User: dbgen.User{
				ID:        user.ID,
				Name:      user.Name,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			},
		})

		return next(c)
	}
}
