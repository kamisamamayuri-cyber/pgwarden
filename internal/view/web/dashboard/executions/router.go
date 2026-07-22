package executions

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
	"github.com/labstack/echo/v4"
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
	parent.GET("/list", h.listExecutionsHandler)
	parent.GET("/:executionID/download", h.downloadExecutionHandler)
	parent.GET("/:executionID/details", h.executionDetailsHandler)
	parent.POST("/:executionID/retry", h.retryExecutionHandler)
	parent.DELETE("/:executionID", h.deleteExecutionHandler)
}
