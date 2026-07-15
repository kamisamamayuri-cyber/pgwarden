package view

import (
	"io/fs"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/logger"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/api"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/middleware"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/static"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

func MountRouter(app *echo.Echo, servs *service.Service) {
	mids := middleware.New(servs)

	// HTML here is highly repetitive (tables, selects) and compresses 10-30x.
	app.Use(echomw.Gzip())

	// Create the base group with the path prefix (if any)
	baseGroup := app.Group(pathutil.GetPathPrefix())

	browserCache := mids.NewBrowserCacheMiddleware(
		middleware.BrowserCacheMiddlewareConfig{
			CacheDuration: time.Hour * 24 * 30,
			ExcludedFiles: []string{"/robots.txt"},
		},
	)

	// Mount static files
	staticFS, err := fs.Sub(static.StaticFs, ".")
	if err != nil {
		logger.FatalError("failed to create static filesystem", logger.KV{"error": err})
	}

	staticGroup := baseGroup.Group("", browserCache)
	staticGroup.StaticFS("/", staticFS)

	apiGroup := baseGroup.Group("/api")
	api.MountRouter(apiGroup, mids, servs)

	webGroup := baseGroup.Group("", mids.InjectReqctx)
	web.MountRouter(webGroup, mids, servs)
}
