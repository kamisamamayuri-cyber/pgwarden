package discovery

import (
	"github.com/labstack/echo/v4"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
)

type handlers struct {
	servs *service.Service
}

func newHandlers(servs *service.Service) *handlers {
	return &handlers{servs: servs}
}

func MountRouter(
	parent *echo.Group, mids *middleware.Middleware, servs *service.Service,
) {
	h := newHandlers(servs)

	parent.GET("", h.indexPageHandler)
	parent.GET("/list", h.listRunsHandler)
	parent.POST("/run", h.runDiscoveryHandler)
	parent.GET("/runs/:runID/details", h.runDetailsHandler)
	parent.GET("/runs/:runID/report", h.runReportHandler)
}
