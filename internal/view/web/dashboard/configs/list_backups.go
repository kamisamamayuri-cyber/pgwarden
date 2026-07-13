package configs

import (
	"net/http"

	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
)

func (h *handlers) listBackupsHandler(c echo.Context) error {
	name := c.Param("name")
	ctx := c.Request().Context()

	backups, err := h.servs.ConfigFilesService.ListBackups(ctx, name)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, backupsList(name, backups))
}

func (h *handlers) restoreBackupHandler(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	if err := h.servs.ConfigFilesService.RestoreBackup(ctx, id); err != nil {
		return echoutil.RenderNodx(c, http.StatusOK, statusError(err.Error()))
	}

	return echoutil.RenderNodx(c, http.StatusOK, statusSuccess("Config rolled back and applied"))
}

func backupsList(configName string, backups []dbgen.ConfigFileBackup) nodx.Node {
	if len(backups) == 0 {
		return nodx.Div(
			nodx.Class("text-base-content/50 text-sm mt-4"),
			component.SpanText("No backups"),
		)
	}

	rows := make([]nodx.Node, 0, len(backups))
	for _, b := range backups {
		restoreURL := pathutil.BuildPath("/dashboard/configs/backups/" + b.ID.String() + "/restore")
		statusID := "config-status-" + configName

		rows = append(rows, nodx.Tr(
			nodx.Td(
				nodx.Text(b.CreatedAt.Format("02.01.2006 15:04:05")),
			),
			nodx.Td(
				nodx.SpanEl(
					nodx.Class("font-mono text-xs text-base-content/60 line-clamp-1"),
					nodx.Text(firstLine(b.Content)),
				),
			),
			nodx.Td(
				nodx.Class("text-right"),
				nodx.Button(
					nodx.Class("btn btn-xs btn-ghost text-warning"),
					htmx.HxPost(restoreURL),
					htmx.HxTarget("#"+statusID),
					htmx.HxSwap("innerHTML"),
					htmx.HxConfirm("Roll back config to this version?"),
					lucide.RotateCcw(nodx.Class("size-3")),
					nodx.Text("Roll back"),
				),
			),
		))
	}

	return nodx.Div(
		nodx.Class("mt-4"),
		nodx.H3(
			nodx.Class("text-sm font-semibold mb-2 text-base-content/70"),
			nodx.Text("Backups (last 10)"),
		),
		component.CardBox(component.CardBoxParams{
			Children: []nodx.Node{
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table table-xs text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(component.SpanText("Date")),
								nodx.Th(component.SpanText("Start")),
								nodx.Th(),
							),
						),
						nodx.Tbody(nodx.Group(rows...)),
					),
				),
			},
		}),
	)
}

func firstLine(s string) string {
	for i, ch := range s {
		if ch == '\n' {
			return s[:i]
		}
	}
	if len(s) > 80 {
		return s[:80] + "…"
	}
	return s
}
