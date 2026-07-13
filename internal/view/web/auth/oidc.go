package auth

import (
	"net/http"
	"time"

	authservice "github.com/kamisamamayuri-cyber/pgwarden/internal/service/auth"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
)

func (h *handlers) oidcStartHandler(c echo.Context) error {
	if !h.servs.AuthService.OidcEnabled() {
		return c.Redirect(http.StatusFound, pathutil.BuildPath("/auth/login"))
	}

	state, err := h.servs.AuthService.NewOidcState()
	if err != nil {
		logger.Error("failed to generate oidc state", logger.KV{"error": err})
		return c.String(http.StatusInternalServerError, i18n.ErrInternalServer)
	}

	authURL, err := h.servs.AuthService.OidcAuthCodeURL(state)
	if err != nil {
		logger.Error("failed to build oidc auth url", logger.KV{"error": err})
		return c.String(http.StatusInternalServerError, i18n.ErrInternalServer)
	}

	c.SetCookie(&http.Cookie{
		Name:     authservice.OidcStateCookieName(),
		Value:    state,
		Path:     pathutil.BuildPath("/"),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   c.Scheme() == "https",
		MaxAge:   int((10 * time.Minute).Seconds()),
	})

	return c.Redirect(http.StatusFound, authURL)
}

func (h *handlers) oidcCallbackHandler(c echo.Context) error {
	if !h.servs.AuthService.OidcEnabled() {
		return c.Redirect(http.StatusFound, pathutil.BuildPath("/auth/login"))
	}

	stateCookie, err := c.Cookie(authservice.OidcStateCookieName())
	if err != nil || stateCookie.Value == "" || stateCookie.Value != c.QueryParam("state") {
		return respondhtmx.ToastError(c, i18n.ErrInvalidOidcState)
	}

	code := c.QueryParam("code")
	if code == "" {
		return respondhtmx.ToastError(c, i18n.ErrMissingOidcCode)
	}

	session, err := h.servs.AuthService.LoginWithOidc(
		c.Request().Context(), code, c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		logger.Error("oidc login failed", logger.KV{
			"ip":    c.RealIP(),
			"ua":    c.Request().UserAgent(),
			"error": err,
		})
		return respondhtmx.ToastError(c, i18n.ErrSsoLoginFailed)
	}

	c.SetCookie(&http.Cookie{
		Name:     authservice.OidcStateCookieName(),
		Value:    "",
		Path:     pathutil.BuildPath("/"),
		HttpOnly: true,
		MaxAge:   -1,
	})

	h.servs.AuthService.SetSessionCookie(c, session.DecryptedToken)
	return respondhtmx.Redirect(c, pathutil.BuildPath("/dashboard"))
}
