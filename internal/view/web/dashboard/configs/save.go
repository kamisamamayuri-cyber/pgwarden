package configs

import (
	"net/http"

	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
)

type configFormDTO struct {
	Content string `form:"content"`
}

func (h *handlers) saveConfigHandler(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	var form configFormDTO
	if err := c.Bind(&form); err != nil {
		return echoutil.RenderNodx(c, http.StatusOK, statusError("Form read error: "+err.Error()))
	}

	if err := h.servs.ConfigFilesService.ValidateAndSave(ctx, name, form.Content); err != nil {
		return echoutil.RenderNodx(c, http.StatusOK, statusError(err.Error()))
	}

	return echoutil.RenderNodx(c, http.StatusOK, statusSuccess("Config saved and applied"))
}

func statusSuccess(msg string) nodx.Node {
	return nodx.Div(
		nodx.Class("alert alert-success flex gap-2 py-2 text-sm"),
		lucide.CircleCheck(nodx.Class("size-4 shrink-0")),
		component.SpanText(msg),
	)
}

func statusError(msg string) nodx.Node {
	return nodx.Div(
		nodx.Class("alert alert-error flex gap-2 py-2 text-sm"),
		lucide.CircleAlert(nodx.Class("size-4 shrink-0")),
		component.SpanText(msg),
	)
}
