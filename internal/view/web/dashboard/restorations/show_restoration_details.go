package restorations

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
)

const htmxStopPollingStatus = 286

func (h *handlers) restorationDetailsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	restorationID, err := uuid.Parse(c.Param("restorationID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	row, err := h.servs.RestorationsService.GetRestorationRow(ctx, restorationID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	view := restorationDetailsViewFromGet(row)
	status := http.StatusOK
	if view.Status != "running" && view.Status != "queued" {
		status = htmxStopPollingStatus
	}
	return echoutil.RenderNodx(c, status, restorationDetailsTable(view))
}
