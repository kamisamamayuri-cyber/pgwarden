package restorations

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
	parent.GET("/list", h.listRestorationsHandler)
	parent.GET("/:restorationID/details", h.restorationDetailsHandler)

	parent.GET("/wizard/step1", h.wizardStep1Handler)
	parent.GET("/wizard/step2", h.wizardStep2Handler)
	parent.GET("/wizard/step3", h.wizardStep3Handler)
	parent.POST("/wizard/run", h.wizardRunHandler)
}
