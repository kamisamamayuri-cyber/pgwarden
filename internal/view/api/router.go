package api

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
	"github.com/labstack/echo/v4"
)

type handlers struct {
	servs *service.Service
}

func MountRouter(
	parent *echo.Group, mids *middleware.Middleware, servs *service.Service,
) {
	v1 := parent.Group("/v1")

	h := &handlers{
		servs: servs,
	}
	v1.GET("/health", h.healthHandler)
	v1.GET("/openapi.yaml", h.openAPIHandler)
	v1.GET("/swagger", h.swaggerUIHandler)

	protected := v1.Group("", mids.RequireAuthAPI)
	protected.GET("/restorations/:restoration_id", h.getRestorationStatusHandler)
	protected.GET("/restores", h.listRestorePresetsHandler)
	protected.GET("/restores/:id/backups/:execution_id", h.getRestoreBackupDownloadHandler)
	protected.GET("/restores/:id/backups", h.listRestoreBackupsHandler)
	protected.GET("/restores/:id/restore", h.getRestoreTargetsHandler)
	protected.POST("/restores/:id/restore", h.runRestoreHandler)
	protected.GET("/restores/:id", h.getRestoreDatabaseHandler)
}
