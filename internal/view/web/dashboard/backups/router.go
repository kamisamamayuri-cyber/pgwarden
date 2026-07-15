package backups

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
	parent.GET("/list", h.listBackupsHandler)
	parent.GET("/create-form", h.createBackupFormHandler)
	parent.GET("/:backupID/edit-form", h.editBackupFormHandler)
	parent.POST("", h.createBackupHandler)
	parent.DELETE("/:backupID", h.deleteBackupHandler)
	parent.POST("/:backupID/edit", h.editBackupHandler)
	parent.POST("/:backupID/run", h.manualRunHandler)
	parent.POST("/:backupID/duplicate", h.duplicateBackupHandler)
}
