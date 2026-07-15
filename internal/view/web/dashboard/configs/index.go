package configs

import (
	"net/http"

	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	alpine "github.com/nodxdev/nodxgo-alpine"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/configfiles"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
)

func (h *handlers) indexPageHandler(c echo.Context) error {
	ctx := c.Request().Context()
	reqCtx := reqctx.GetCtx(c)

	presets, err := h.servs.ConfigFilesService.GetConfig(ctx, configfiles.NameRestorePresets)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	disc, err := h.servs.ConfigFilesService.GetConfig(ctx, configfiles.NameDiscovery)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	presetsBackups, err := h.servs.ConfigFilesService.ListBackups(ctx, configfiles.NameRestorePresets)
	if err != nil {
		presetsBackups = nil
	}

	discBackups, err := h.servs.ConfigFilesService.ListBackups(ctx, configfiles.NameDiscovery)
	if err != nil {
		discBackups = nil
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, presets, disc, presetsBackups, discBackups))
}

func indexPage(
	reqCtx reqctx.Ctx,
	presets, disc dbgen.ConfigFile,
	presetsBackups, discBackups []dbgen.ConfigFileBackup,
) nodx.Node {
	content := []nodx.Node{
		nodx.Div(
			nodx.Class("flex justify-between items-start gap-4"),
			nodx.Div(
				component.H1Text("Configs"),
				component.PText("Edit application configuration files. Available to administrators only."),
			),
		),

		nodx.Div(
			nodx.Class("mt-6"),
			nodx.Div(
				nodx.Role("tablist"),
				nodx.Class("tabs tabs-lifted"),

				// Tab: restore-presets
				nodx.Input(
					nodx.Type("radio"),
					nodx.Name("config-tab"),
					nodx.Role("tab"),
					nodx.Class("tab"),
					nodx.Attr("aria-label", "Restore presets"),
					nodx.Checked(""),
				),
				nodx.Div(
					nodx.Role("tabpanel"),
					nodx.Class("tab-content bg-base-100 border-base-300 rounded-box p-4"),
					configEditor(configfiles.NameRestorePresets, presets, presetsBackups),
				),

				// Tab: discovery
				nodx.Input(
					nodx.Type("radio"),
					nodx.Name("config-tab"),
					nodx.Role("tab"),
					nodx.Class("tab"),
					nodx.Attr("aria-label", "Discovery"),
				),
				nodx.Div(
					nodx.Role("tabpanel"),
					nodx.Class("tab-content bg-base-100 border-base-300 rounded-box p-4"),
					configEditor(configfiles.NameDiscovery, disc, discBackups),
				),
			),
		),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Configs",
		Body:  content,
	})
}

func configEditor(name string, cfg dbgen.ConfigFile, backups []dbgen.ConfigFileBackup) nodx.Node {
	formID := "config-form-" + name
	textareaID := "config-textarea-" + name
	statusID := "config-status-" + name
	backupsID := "config-backups-" + name

	saveURL := pathutil.BuildPath("/dashboard/configs/" + name + "/save")
	backupsURL := pathutil.BuildPath("/dashboard/configs/" + name + "/backups")

	alpineID := "alpineConfigEditor('" + textareaID + "')"

	return nodx.Div(
		nodx.Class("space-y-4"),
		alpine.XData(alpineID),
		alpine.XInit("init()"),

		nodx.FormEl(
			nodx.Id(formID),
			nodx.Class("space-y-3"),

			nodx.Div(
				nodx.Class("form-control"),
				nodx.LabelEl(
					nodx.Class("label py-1"),
					component.SpanText("YAML configuration"),
					nodx.SpanEl(
						nodx.Class("label-text-alt text-base-content/50"),
						nodx.Text("Updated: "+cfg.UpdatedAt.Format("02.01.2006 15:04:05")),
					),
				),
				nodx.Textarea(
					nodx.Id(textareaID),
					nodx.Name("content"),
					nodx.Class("textarea textarea-bordered font-mono text-sm w-full h-96 resize-y"),
					nodx.Text(cfg.Content),
				),
				// Inline real-time syntax status
				nodx.Div(
					nodx.Class("mt-1 text-sm flex items-center gap-1"),
					nodx.Attr("x-show", "status === 'ok'"),
					nodx.Attr("x-cloak", ""),
					nodx.Class("text-success"),
					lucide.CircleCheck(nodx.Class("size-4 inline")),
					nodx.SpanEl(nodx.Text(" YAML is valid")),
				),
				nodx.Div(
					nodx.Class("mt-1 text-sm flex items-start gap-1"),
					nodx.Attr("x-show", "status === 'error'"),
					nodx.Attr("x-cloak", ""),
					nodx.Class("text-error"),
					lucide.CircleAlert(nodx.Class("size-4 inline shrink-0 mt-0.5")),
					nodx.SpanEl(
						nodx.Attr("x-text", "errorMsg"),
					),
				),
			),

			// Status block (HTMX target for server validate/save responses)
			nodx.Div(
				nodx.Id(statusID),
			),

			nodx.Div(
				nodx.Class("flex gap-2 justify-end"),

				// Save button
				nodx.Button(
					nodx.Type("button"),
					nodx.Class("btn btn-primary btn-sm"),
					htmx.HxPost(saveURL),
					htmx.HxInclude("#"+formID),
					htmx.HxTarget("#"+statusID),
					htmx.HxSwap("innerHTML"),
					htmx.HxOn("htmx:after-request", "if(event.detail.successful && event.detail.xhr.status===200) { htmx.trigger('#"+backupsID+"', 'refresh-backups'); }"),
					lucide.Save(nodx.Class("size-4")),
					nodx.Text("Save"),
				),
			),
		),

		// Backups list
		nodx.Div(
			nodx.Id(backupsID),
			htmx.HxGet(backupsURL),
			htmx.HxTrigger("load, refresh-backups"),
			htmx.HxSwap("innerHTML"),
			component.SkeletonTr(3),
		),
	)
}
