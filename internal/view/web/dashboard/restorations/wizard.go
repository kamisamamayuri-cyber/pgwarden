package restorations

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auditlogs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

const wizardModalID = "new-restore-wizard"

// ── Step 1: select preset ────────────────────────────────────────────────────

func (h *handlers) wizardStep1Handler(c echo.Context) error {
	ctx := c.Request().Context()
	access := reqctx.GetCtx(c).Access
	rbacEnabled := h.servs.RbacService.Enabled()

	catalog, err := h.servs.RestorationsService.ListRestoreCatalog(ctx)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var presets []restorations.RestoreDatabaseInfo
	for _, db := range catalog.Databases {
		if !rbacEnabled || access.CanViewPreset(db.ID) {
			db.CanExecute = !rbacEnabled || access.CanExecutePreset(db.ID)
			presets = append(presets, db)
		}
	}

	return echoutil.RenderNodx(c, http.StatusOK, wizardStep1Content(presets))
}

func wizardStep1Content(presets []restorations.RestoreDatabaseInfo) nodx.Node {
	if len(presets) == 0 {
		return nodx.Div(
			nodx.Class("py-8 text-center text-base-content/60"),
			nodx.Text("No restore presets available."),
		)
	}

	items := make([]nodx.Node, 0, len(presets))
	for i, p := range presets {
		items = append(items, wizardPresetItem(p, i == 0))
	}

	return nodx.FormEl(
		nodx.Id("wizard-form"),
		htmx.HxGet(pathutil.BuildPath("/dashboard/restorations/wizard/step2")),
		htmx.HxTarget("#wizard-slot"),
		htmx.HxSwap("innerHTML"),
		htmx.HxInclude("#wizard-form"),

		nodx.P(
			nodx.Class("text-sm text-base-content/60 mb-3"),
			nodx.Text("Select the restore preset to use:"),
		),

		nodx.Div(
			nodx.Class("space-y-2 mb-4"),
			nodx.Group(items...),
		),

		wizardFooter(
			nodx.Button(
				nodx.Type("button"),
				nodx.Class("btn btn-ghost btn-sm"),
				nodx.Attr("onClick", fmt.Sprintf("window.dispatchEvent(new Event('%s_close'));", wizardModalID)),
				nodx.Text("Cancel"),
			),
			nodx.Button(
				nodx.Type("submit"),
				nodx.Class("btn btn-primary btn-sm"),
				nodx.Text("Next"),
				lucide.ChevronRight(nodx.Class("size-4")),
			),
		),
	)
}

func wizardPresetItem(p restorations.RestoreDatabaseInfo, checked bool) nodx.Node {
	sourceInfo := fmt.Sprintf("%s:%d / %s", p.Source.Host, p.Source.Port, p.Source.Database)

	return nodx.LabelEl(
		nodx.Class("flex items-start gap-3 p-3 rounded-lg border border-base-300 cursor-pointer hover:bg-base-300 has-[input:checked]:border-primary has-[input:checked]:bg-primary/5"),
		nodx.Input(
			nodx.Type("radio"),
			nodx.Name("preset_id"),
			nodx.Value(p.ID),
			nodx.Class("radio radio-primary radio-sm mt-0.5 flex-shrink-0"),
			nodx.If(checked, nodx.Attr("checked", "")),
		),
		nodx.Div(
			nodx.Class("flex-1 min-w-0"),
			nodx.Div(
				nodx.Class("font-medium text-sm"),
				nodx.Text(p.Title),
			),
			nodx.If(p.Description != "", nodx.Div(
				nodx.Class("text-xs text-base-content/60 mt-0.5"),
				nodx.Text(p.Description),
			)),
			nodx.Div(
				nodx.Class("text-xs text-base-content/50 mt-1 font-mono"),
				nodx.Text("Source: "+sourceInfo),
			),
		),
	)
}

// ── Step 2: select environment ───────────────────────────────────────────────

type wizardStep2Query struct {
	PresetID string `query:"preset_id"`
}

func (h *handlers) wizardStep2Handler(c echo.Context) error {
	ctx := c.Request().Context()
	access := reqctx.GetCtx(c).Access
	rbacEnabled := h.servs.RbacService.Enabled()

	var q wizardStep2Query
	if err := c.Bind(&q); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if q.PresetID == "" {
		return respondhtmx.ToastError(c, "preset_id is required")
	}
	if rbacEnabled && !access.CanViewPreset(q.PresetID) {
		return respondhtmx.ToastError(c, "access denied")
	}

	info, err := h.servs.RestorationsService.GetRestoreTargets(ctx, q.PresetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return respondhtmx.ToastError(c, "preset not found")
		}
		return respondhtmx.ToastError(c, err.Error())
	}

	canExecute := !rbacEnabled || access.CanExecutePreset(q.PresetID)
	return echoutil.RenderNodx(c, http.StatusOK, wizardStep2Content(q.PresetID, info, canExecute))
}

func wizardStep2Content(
	presetID string,
	info restorations.RestoreTargetsInfo,
	canExecute bool,
) nodx.Node {
	if len(info.Environments) == 0 {
		return nodx.Div(
			nodx.Class("py-8 text-center text-base-content/60"),
			nodx.Text("No environments configured for this preset."),
		)
	}

	items := make([]nodx.Node, 0, len(info.Environments))
	for i, env := range info.Environments {
		items = append(items, wizardEnvItem(env, i == 0))
	}

	return nodx.FormEl(
		nodx.Id("wizard-form"),
		htmx.HxGet(pathutil.BuildPath("/dashboard/restorations/wizard/step3")),
		htmx.HxTarget("#wizard-slot"),
		htmx.HxSwap("innerHTML"),
		htmx.HxInclude("#wizard-form"),

		nodx.Input(nodx.Type("hidden"), nodx.Name("preset_id"), nodx.Value(presetID)),

		nodx.P(
			nodx.Class("text-sm text-base-content/60 mb-3"),
			nodx.Text("Select the target environment to restore into:"),
		),

		nodx.If(!canExecute, nodx.Div(
			nodx.Class("alert alert-warning text-sm mb-3"),
			nodx.Text("You have view-only access to this preset and cannot launch restores."),
		)),

		nodx.Div(
			nodx.Class("space-y-2 mb-4"),
			nodx.Group(items...),
		),

		wizardFooter(
			nodx.Button(
				nodx.Type("button"),
				nodx.Class("btn btn-ghost btn-sm"),
				htmx.HxGet(pathutil.BuildPath("/dashboard/restorations/wizard/step1")),
				htmx.HxTarget("#wizard-slot"),
				htmx.HxSwap("innerHTML"),
				lucide.ChevronLeft(nodx.Class("size-4")),
				nodx.Text("Back"),
			),
			nodx.Button(
				nodx.Type("submit"),
				nodx.Class("btn btn-primary btn-sm"),
				nodx.If(!canExecute, nodx.Attr("disabled", "")),
				nodx.Text("Next"),
				lucide.ChevronRight(nodx.Class("size-4")),
			),
		),
	)
}

func wizardEnvItem(env restorations.RestoreEnvironmentInfo, checked bool) nodx.Node {
	targetInfo := fmt.Sprintf("%s:%d / %s", env.Target.Host, env.Target.Port, env.Target.Database)

	return nodx.LabelEl(
		nodx.Class("flex items-start gap-3 p-3 rounded-lg border border-base-300 cursor-pointer hover:bg-base-300 has-[input:checked]:border-primary has-[input:checked]:bg-primary/5"),
		nodx.Input(
			nodx.Type("radio"),
			nodx.Name("environment"),
			nodx.Value(env.Environment),
			nodx.Class("radio radio-primary radio-sm mt-0.5 flex-shrink-0"),
			nodx.If(checked, nodx.Attr("checked", "")),
		),
		nodx.Div(
			nodx.Class("flex-1 min-w-0"),
			nodx.Div(
				nodx.Class("font-medium text-sm capitalize"),
				nodx.Text(env.Environment),
			),
			nodx.Div(
				nodx.Class("text-xs text-base-content/50 mt-1 font-mono"),
				nodx.Text(targetInfo),
			),
		),
	)
}

// ── Step 3: select backup ────────────────────────────────────────────────────

type wizardStep3Query struct {
	PresetID    string `query:"preset_id"`
	Environment string `query:"environment"`
}

func (h *handlers) wizardStep3Handler(c echo.Context) error {
	ctx := c.Request().Context()
	access := reqctx.GetCtx(c).Access
	rbacEnabled := h.servs.RbacService.Enabled()

	var q wizardStep3Query
	if err := c.Bind(&q); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if q.PresetID == "" || q.Environment == "" {
		return respondhtmx.ToastError(c, "preset_id and environment are required")
	}
	if rbacEnabled && !access.CanViewPreset(q.PresetID) {
		return respondhtmx.ToastError(c, "access denied")
	}

	backupList, err := h.servs.RestorationsService.ListRestoreBackups(ctx, q.PresetID, 20)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return respondhtmx.ToastError(c, "preset not found")
		}
		return respondhtmx.ToastError(c, err.Error())
	}

	canExecute := !rbacEnabled || access.CanExecutePreset(q.PresetID)
	return echoutil.RenderNodx(c, http.StatusOK, wizardStep3Content(q.PresetID, q.Environment, backupList, canExecute))
}

func wizardStep3Content(
	presetID, environment string,
	backupList restorations.RestoreBackupList,
	canExecute bool,
) nodx.Node {
	if len(backupList.Backups) == 0 {
		return nodx.Div(
			nodx.Class("space-y-4"),
			nodx.Div(
				nodx.Class("py-8 text-center text-base-content/60"),
				nodx.Text("No successful backups found for this preset."),
			),
			wizardFooter(
				nodx.Button(
					nodx.Type("button"),
					nodx.Class("btn btn-ghost btn-sm"),
					htmx.HxGet(pathutil.BuildPath(
						fmt.Sprintf("/dashboard/restorations/wizard/step2?preset_id=%s", presetID),
					)),
					htmx.HxTarget("#wizard-slot"),
					htmx.HxSwap("innerHTML"),
					lucide.ChevronLeft(nodx.Class("size-4")),
					nodx.Text("Back"),
				),
				nodx.SpanEl(),
			),
		)
	}

	rows := make([]nodx.Node, 0, len(backupList.Backups))
	for i, b := range backupList.Backups {
		isLatest := i == 0
		rows = append(rows, nodx.Tr(
			nodx.Td(
				nodx.Input(
					nodx.Type("radio"),
					nodx.Name("execution_id"),
					nodx.Value(b.ExecutionID.String()),
					nodx.Class("radio radio-primary radio-sm"),
					nodx.If(isLatest, nodx.Attr("checked", "")),
				),
			),
			nodx.Td(
				component.SpanText(b.FinishedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty)),
				nodx.If(isLatest, nodx.SpanEl(
					nodx.Class("badge badge-primary badge-sm ml-2"),
					nodx.Text("latest"),
				)),
			),
			nodx.Td(component.SpanText(b.PgVersion)),
			nodx.Td(component.SpanText(wizardFormatFileSize(b.FileSize))),
		))
	}

	return nodx.FormEl(
		nodx.Id("wizard-form"),
		htmx.HxPost(pathutil.BuildPath("/dashboard/restorations/wizard/run")),
		htmx.HxTarget("#wizard-slot"),
		htmx.HxSwap("innerHTML"),
		htmx.HxInclude("#wizard-form"),
		htmx.HxConfirm(fmt.Sprintf(
			"Restore into '%s'? This will overwrite the target database. Continue?",
			environment,
		)),

		nodx.Input(nodx.Type("hidden"), nodx.Name("preset_id"), nodx.Value(presetID)),
		nodx.Input(nodx.Type("hidden"), nodx.Name("environment"), nodx.Value(environment)),

		nodx.P(
			nodx.Class("text-sm text-base-content/60 mb-3"),
			nodx.Text("Select the backup to restore from (latest is pre-selected):"),
		),

		nodx.Div(
			nodx.Class("alert alert-warning text-sm mb-3"),
			lucide.TriangleAlert(nodx.Class("size-4 shrink-0")),
			nodx.SpanEl(nodx.Text("This operation will overwrite all data in the target database.")),
		),

		nodx.Div(
			nodx.Class("overflow-x-auto max-h-64 mb-4"),
			nodx.Table(
				nodx.Class("table table-sm text-nowrap"),
				nodx.Thead(
					nodx.Tr(
						nodx.Th(),
						nodx.Th(component.SpanText("Finished at")),
						nodx.Th(component.SpanText("PG Version")),
						nodx.Th(component.SpanText("Size")),
					),
				),
				nodx.Tbody(nodx.Group(rows...)),
			),
		),

		wizardFooter(
			nodx.Button(
				nodx.Type("button"),
				nodx.Class("btn btn-ghost btn-sm"),
				htmx.HxGet(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/restorations/wizard/step2?preset_id=%s", presetID),
				)),
				htmx.HxTarget("#wizard-slot"),
				htmx.HxSwap("innerHTML"),
				lucide.ChevronLeft(nodx.Class("size-4")),
				nodx.Text("Back"),
			),
			nodx.Button(
				nodx.Type("submit"),
				nodx.Class("btn btn-error btn-sm"),
				nodx.If(!canExecute, nodx.Attr("disabled", "")),
				lucide.Play(nodx.Class("size-4")),
				nodx.Text("Launch Restore"),
			),
		),
	)
}

// ── Run handler ──────────────────────────────────────────────────────────────

type wizardRunForm struct {
	PresetID    string `form:"preset_id"`
	Environment string `form:"environment"`
	ExecutionID string `form:"execution_id"`
}

func (h *handlers) wizardRunHandler(c echo.Context) error {
	ctx := c.Request().Context()
	access := reqctx.GetCtx(c).Access
	rbacEnabled := h.servs.RbacService.Enabled()

	var form wizardRunForm
	if err := c.Bind(&form); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if form.PresetID == "" || form.Environment == "" {
		return respondhtmx.ToastError(c, "preset_id and environment are required")
	}

	if rbacEnabled {
		if !access.CanViewPreset(form.PresetID) {
			return respondhtmx.ToastError(c, "access denied")
		}
		if !access.CanExecutePreset(form.PresetID) {
			return respondhtmx.ToastError(c, "forbidden: you cannot execute restores for this preset")
		}
	}

	params := restorations.RestoreStartParams{
		Environment: form.Environment,
	}
	if form.ExecutionID != "" {
		id, err := uuid.Parse(form.ExecutionID)
		if err != nil {
			return respondhtmx.ToastError(c, "invalid execution_id")
		}
		params.ExecutionID = &id
	}

	result, err := h.servs.RestorationsService.StartRestore(ctx, form.PresetID, params)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return respondhtmx.ToastError(c, "preset not found")
		case errors.Is(err, restorations.ErrRestoreEnvironmentRequired):
			return respondhtmx.ToastError(c, "environment is required")
		case errors.Is(err, restorations.ErrRestoreEnvironmentNotFound):
			return respondhtmx.ToastError(c, "environment not found")
		case errors.Is(err, restorations.ErrRestoreActionNotReady):
			return respondhtmx.ToastError(c, "no backup available for this preset yet")
		case errors.Is(err, restorations.ErrRestoreAlreadyRunning):
			return respondhtmx.ToastError(c, "a restore is already running for this target")
		case errors.Is(err, restorations.ErrRestoreBackupNotFound):
			return respondhtmx.ToastError(c, "backup not found")
		default:
			return respondhtmx.ToastError(c, err.Error())
		}
	}

	userEmail := reqctx.GetCtx(c).User.Email
	execID := result.ExecutionID
	env := result.Environment
	h.servs.AuditLogsService.Log(ctx, auditlogs.Entry{
		UserEmail:   userEmail,
		Action:      "run_restore",
		PresetID:    form.PresetID,
		PresetTitle: result.Title,
		ExecutionID: &execID,
		Environment: &env,
		Source:      "ui",
	})

	c.Response().Header().Set("HX-Trigger", `{"ctm_toast_success": "Restore queued!"}`)
	return echoutil.RenderNodx(c, http.StatusOK, wizardSuccessContent(result))
}

func wizardSuccessContent(result restorations.RestoreStartResult) nodx.Node {
	return nodx.Div(
		nodx.Class("py-8 text-center space-y-3"),
		nodx.Div(
			nodx.Class("flex justify-center text-success"),
			lucide.CircleCheck(nodx.Class("size-16")),
		),
		nodx.Div(
			nodx.Class("text-lg font-bold"),
			nodx.Text("Restore queued!"),
		),
		nodx.Div(
			nodx.Class("text-sm text-base-content/60"),
			nodx.Text(result.Title),
		),
		nodx.Div(
			nodx.Class("mt-4 flex justify-center gap-2"),
			nodx.Button(
				nodx.Type("button"),
				nodx.Class("btn btn-ghost btn-sm"),
				nodx.Attr("onClick", fmt.Sprintf("window.dispatchEvent(new Event('%s_close'));", wizardModalID)),
				nodx.Text("Close"),
			),
			nodx.A(
				nodx.Class("btn btn-primary btn-sm"),
				nodx.Href(pathutil.BuildPath("/dashboard/restorations")),
				nodx.Text("View Restorations"),
			),
		),
	)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func wizardFooter(left, right nodx.Node) nodx.Node {
	return nodx.Div(
		nodx.Class("flex justify-between items-center pt-2 border-t border-base-300"),
		left,
		right,
	)
}

func wizardFormatFileSize(bytes int64) string {
	if bytes == 0 {
		return ""
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
