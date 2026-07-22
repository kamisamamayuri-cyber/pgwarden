package auth

import (
	"net/http"
	"time"

	authservice "github.com/kamisamamayuri-cyber/pgwarden/internal/service/auth"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
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
		return echoutil.RenderNodx(c, http.StatusUnauthorized, oidcErrorPage(i18n.ErrInvalidOidcState))
	}

	code := c.QueryParam("code")
	if code == "" {
		return echoutil.RenderNodx(c, http.StatusBadRequest, oidcErrorPage(i18n.ErrMissingOidcCode))
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
		return echoutil.RenderNodx(c, http.StatusUnauthorized, oidcErrorPage(i18n.ErrSsoLoginFailed))
	}

	c.SetCookie(&http.Cookie{
		Name:     authservice.OidcStateCookieName(),
		Value:    "",
		Path:     pathutil.BuildPath("/"),
		HttpOnly: true,
		MaxAge:   -1,
	})

	h.servs.AuthService.SetSessionCookie(c, session.DecryptedToken)
	dashboardURL := pathutil.BuildPath("/dashboard")
	// HX-Redirect for HTMX-initiated flows (XHR chain); meta+JS for direct browser navigation
	htmx.ServerSetRedirect(c.Response().Header(), dashboardURL)
	return echoutil.RenderNodx(c, http.StatusOK, oidcRedirectPage(dashboardURL))
}

func oidcRedirectPage(url string) nodx.Node {
	return nodx.Group(
		nodx.DocType(),
		nodx.Html(
			nodx.Lang(component.AppPageLang),
			nodx.Head(
				nodx.Meta(nodx.Charset("utf-8")),
				nodx.Meta(nodx.HttpEquiv("refresh"), nodx.Content("0;url="+url)),
			),
			nodx.Body(
				nodx.Script(nodx.Rawf("window.location.href='%s';", url)),
			),
		),
	)
}

func oidcErrorPage(message string) nodx.Node {
	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex items-center gap-3 mb-4"),
			lucide.ShieldX(nodx.Class("size-8 text-error shrink-0")),
			component.H1Text("Authentication failed"),
		),
		nodx.Div(
			nodx.Role("alert"),
			nodx.Class("alert alert-error mb-5"),
			lucide.CircleAlert(nodx.Class("size-5 shrink-0")),
			nodx.SpanEl(nodx.Text(message)),
		),
		nodx.Div(
			nodx.Class("flex justify-end"),
			nodx.A(
				nodx.Class("btn btn-primary"),
				nodx.Href(pathutil.BuildPath("/auth/oidc/start")),
				lucide.LogIn(nodx.Class("size-4")),
				nodx.Text("Sign in again"),
			),
		),
	}

	return layout.Auth(layout.AuthParams{
		Title: "Authentication Error",
		Body:  content,
	})
}
