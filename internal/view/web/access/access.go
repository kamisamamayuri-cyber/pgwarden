package access

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/labstack/echo/v4"
)

func FromContext(c echo.Context, servs *service.Service) rbac.Access {
	rc := reqctx.GetCtx(c)
	return servs.RbacService.Access(rc.Groups, rc.FullAccess)
}

func ForbiddenHTML(c echo.Context) error {
	return c.String(http.StatusForbidden, "Access denied")
}

func ForbiddenJSON(c echo.Context) error {
	return c.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
}

func ForbiddenHTMX(c echo.Context) error {
	return ForbiddenHTML(c)
}
