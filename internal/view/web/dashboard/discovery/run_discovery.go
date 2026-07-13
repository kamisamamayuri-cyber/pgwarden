package discovery

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	discoveryservice "github.com/kamisamamayuri-cyber/pgwarden/internal/service/discovery"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
)

func (h *handlers) runDiscoveryHandler(c echo.Context) error {
	queryData := listQueryData{
		Level:    c.QueryParam("level"),
		Event:    c.QueryParam("event"),
		Host:     c.QueryParam("host"),
		Database: c.QueryParam("database"),
		Page:     1,
	}
	if port := c.QueryParam("port"); port != "" {
		parsedPort, err := strconv.Atoi(port)
		if err != nil {
			return respondhtmx.ToastError(c, err.Error())
		}
		queryData.Port = parsedPort
	}
	if err := validate.Struct(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if running, err := h.servs.DiscoveryService.Running(c.Request().Context()); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	} else if running {
		return respondhtmx.ToastError(c, "Discovery is already running, wait for the current run to finish")
	}

	go func() {
		_, err := h.servs.DiscoveryService.Run(context.Background())
		if err != nil && !errors.Is(err, discoveryservice.ErrDiscoveryAlreadyRunning) {
			// Best-effort background run; errors are visible in discovery log.
			_ = err
		}
	}()
	time.Sleep(200 * time.Millisecond)

	q := filterQuery{
		Level:    queryData.Level,
		Event:    queryData.Event,
		Host:     queryData.Host,
		Port:     queryData.Port,
		Database: queryData.Database,
	}
	pagination, runs, err := h.servs.DiscoveryService.PaginateRuns(
		c.Request().Context(),
		paginateParamsFromQuery(queryData),
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, listRuns(q, pagination, runs))
}
