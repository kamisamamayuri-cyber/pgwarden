package dashboard

import (
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/about"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/backups"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/configs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/databases"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/destinations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/discovery"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/executions"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/logs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/profile"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/summary"
	"github.com/labstack/echo/v4"
)

func MountRouter(
	parent *echo.Group, mids *middleware.Middleware, servs *service.Service,
) {
	parent.GET("/health-button", healthButtonHandler(servs))
	parent.GET("/version-button", versionButtonHandler(servs))

	summary.MountRouter(parent.Group(""), mids, servs)
	databases.MountRouter(parent.Group("/databases"), mids, servs)
	configs.MountRouter(parent.Group("/configs", mids.RequireManageApp), mids, servs)
	destinations.MountRouter(parent.Group("/destinations", mids.RequireManageApp), mids, servs)
	discovery.MountRouter(parent.Group("/discovery", mids.RequireManageApp), mids, servs)
	logs.MountRouter(parent.Group("/logs", mids.RequireManageApp), mids, servs)
	backups.MountRouter(parent.Group("/backups"), mids, servs)
	executions.MountRouter(parent.Group("/jobs"), mids, servs)
	restorations.MountRouter(parent.Group("/restorations", mids.RequireRestoreAccess), mids, servs)
	profile.MountRouter(parent.Group("/profile"), mids, servs)
	about.MountRouter(parent.Group("/about"), mids, servs)
}
