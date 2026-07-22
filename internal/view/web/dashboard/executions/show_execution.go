package executions

import (
	"database/sql"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/auditlogs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/logtail"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) downloadExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	execution, err := h.servs.ExecutionsService.GetExecution(ctx, executionID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	database, err := h.servs.DatabasesService.GetDatabase(ctx, execution.DatabaseID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanViewPbwName(database.Name) {
		return webaccess.ForbiddenHTML(c)
	}

	isLocal, link, err := h.servs.ExecutionsService.GetExecutionDownloadLinkOrPath(
		ctx, executionID,
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, execution.BackupID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	h.servs.AuditLogsService.Log(ctx, auditlogs.Entry{
		UserEmail:   reqctx.GetCtx(c).User.Email,
		Action:      "download_dump",
		PresetID:    backup.ID.String(),
		PresetTitle: backup.Name,
		ExecutionID: &executionID,
		Source:      "ui",
	})

	if isLocal {
		return c.Attachment(link, filepath.Base(link))
	}

	return c.Redirect(http.StatusFound, link)
}

func (h *handlers) retryExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	execution, err := h.servs.ExecutionsService.GetExecution(ctx, executionID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if execution.Status != "failed" {
		return respondhtmx.ToastError(c, "only failed executions can be retried")
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, execution.BackupID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanExecutePbwName(backup.Name) {
		return webaccess.ForbiddenHTMX(c)
	}

	if err := h.servs.ExecutionsService.EnqueueExecution(ctx, backup.ID); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	h.servs.AuditLogsService.Log(ctx, auditlogs.Entry{
		UserEmail:   reqctx.GetCtx(c).User.Email,
		Action:      "run_backup",
		PresetID:    backup.ID.String(),
		PresetTitle: backup.Name,
		Source:      "ui",
	})

	return respondhtmx.ToastSuccess(c, "Backup re-queued, see details on the executions page")
}

func retryExecutionButton(execution dbgen.ExecutionsServicePaginateExecutionsRow) nodx.Node {
	return component.OptionsDropdownButton(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/jobs/%s/retry", execution.ID))),
		htmx.HxDisabledELT("this"),
		htmx.HxConfirm("Retry this backup? A new execution will be queued for the same backup."),
		lucide.RotateCcw(),
		component.SpanText("Retry"),
	)
}

func (h *handlers) executionDetailsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	row, err := h.servs.ExecutionsService.GetExecutionDetails(ctx, executionID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanViewPbwName(row.DatabaseName) {
		return webaccess.ForbiddenHTMX(c)
	}

	view := executionDetailsViewFromGetDetails(row)
	status := http.StatusOK
	if view.Status != "running" && view.Status != "queued" {
		status = htmxStopPollingStatus
	}
	return echoutil.RenderNodx(c, status, renderExecutionDetailsBody(view, access))
}

type executionDetailsView struct {
	ID              uuid.UUID
	Status          string
	BackupName      string
	DatabaseName    string
	DestinationName sql.NullString
	BackupIsLocal   bool
	Message         sql.NullString
	LogTail         sql.NullString
	StartedAt       time.Time
	FinishedAt      sql.NullTime
	DeletedAt       sql.NullTime
	FileSize        sql.NullInt64
}

func executionDetailsViewFromPaginate(
	row dbgen.ExecutionsServicePaginateExecutionsRow,
) executionDetailsView {
	return executionDetailsView{
		ID:              row.ID,
		Status:          row.Status,
		BackupName:      row.BackupName,
		DatabaseName:    row.DatabaseName,
		DestinationName: row.DestinationName,
		BackupIsLocal:   row.BackupIsLocal,
		Message:         row.Message,
		LogTail:         row.LogTail,
		StartedAt:       row.StartedAt,
		FinishedAt:      row.FinishedAt,
		DeletedAt:       row.DeletedAt,
		FileSize:        row.FileSize,
	}
}

func executionDetailsViewFromGetDetails(
	row dbgen.ExecutionsServiceGetExecutionDetailsRow,
) executionDetailsView {
	return executionDetailsView{
		ID:              row.ID,
		Status:          row.Status,
		BackupName:      row.BackupName,
		DatabaseName:    row.DatabaseName,
		DestinationName: row.DestinationName,
		BackupIsLocal:   row.BackupIsLocal,
		Message:         row.Message,
		LogTail:         row.LogTail,
		StartedAt:       row.StartedAt,
		FinishedAt:      row.FinishedAt,
		DeletedAt:       row.DeletedAt,
		FileSize:        row.FileSize,
	}
}

func executionModalID(id uuid.UUID) string { return "execution-details-" + id.String() }
func executionDetailsLoadingID(id uuid.UUID) string {
	return "execution-details-loading-" + id.String()
}

func renderExecutionDetails(v executionDetailsView, access rbac.Access, poll bool) nodx.Node {
	detailsID := "execution-details-content-" + v.ID.String()
	logID := executionLogID(v.ID)

	body := renderExecutionDetailsBody(v, access)

	nodes := []nodx.Node{nodx.Id(detailsID), nodx.Class("overflow-x-auto"), body}
	if poll {
		openEvent := executionModalID(v.ID) + "_open from:window"
		nodes = append([]nodx.Node{
			htmx.HxGet(pathutil.BuildPath(fmt.Sprintf("/dashboard/jobs/%s/details", v.ID))),
			htmx.HxTrigger(openEvent + ", every 3s"),
			htmx.HxSwap("innerHTML"),
			htmx.HxIndicator("#" + executionDetailsLoadingID(v.ID)),
			nodx.Attr("hx-on:htmx:before-swap",
				"let el=this.querySelector('#"+logID+"'); if(el) this.dataset.logScroll=el.scrollTop"),
			nodx.Attr("hx-on:htmx:after-swap",
				"let el=this.querySelector('#"+logID+"'); if(el && this.dataset.logScroll) el.scrollTop=this.dataset.logScroll"),
		}, nodes...)
	}

	return nodx.Div(nodes...)
}

func executionLogID(id uuid.UUID) string {
	return "execution-log-" + id.String()
}

func renderExecutionDetailsBody(v executionDetailsView, access rbac.Access) nodx.Node {
	destCell := nodx.Node(component.PrettyDestinationName(v.BackupIsLocal, v.DestinationName))
	if !access.CanManageApp() {
		destCell = component.SpanText("S3")
	}

	logLines := logtail.Parse(v.LogTail.String)

	actionNodes := []nodx.Node{}
	if v.Status == "success" {
		if access.CanManageApp() {
			actionNodes = append(actionNodes, deleteExecutionButton(v.ID))
		}
		if access.CanViewPbwName(v.DatabaseName) {
			actionNodes = append(actionNodes, nodx.A(
				nodx.Href(pathutil.BuildPath(fmt.Sprintf("/dashboard/jobs/%s/download", v.ID))),
				nodx.Target("_blank"),
				nodx.Class("btn btn-primary"),
				component.SpanText(i18n.BtnDownload),
				lucide.Download(),
			))
		}
	}

	table := nodx.Table(
		nodx.Class("table [&_th]:text-nowrap"),
		nodx.Tr(
			nodx.Th(component.SpanText("ID")),
			nodx.Td(component.SpanText(v.ID.String())),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelStatus)),
			nodx.Td(component.StatusBadge(v.Status)),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelDatabase)),
			nodx.Td(component.SpanText(v.DatabaseName)),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelDestination)),
			nodx.Td(destCell),
		),
		nodx.If(
			v.Message.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelMessage)),
				nodx.Td(
					nodx.Class("break-all"),
					component.SpanText(v.Message.String),
				),
			),
		),
		nodx.If(
			len(logLines) > 0,
			nodx.Tr(
				nodx.Th(component.SpanText("Log (last lines)")),
				nodx.Td(component.LogTailBox(logLines, nodx.Id(executionLogID(v.ID)))),
			),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelStartedAt)),
			nodx.Td(component.SpanText(
				v.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
		),
		nodx.If(
			v.FinishedAt.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelFinishedAt)),
				nodx.Td(component.SpanText(
					v.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
			),
		),
		nodx.If(
			v.FinishedAt.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelDuration)),
				nodx.Td(component.SpanText(
					v.FinishedAt.Time.Sub(v.StartedAt).String(),
				)),
			),
		),
		nodx.If(
			v.DeletedAt.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText("Deleted at")),
				nodx.Td(component.SpanText(
					v.DeletedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
			),
		),
		nodx.If(
			v.FileSize.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelFileSize)),
				nodx.Td(component.PrettyFileSize(v.FileSize)),
			),
		),
	)

	return nodx.Group(
		table,
		nodx.If(
			len(actionNodes) > 0,
			nodx.Div(
				nodx.Class("flex justify-end items-center space-x-2 mt-2"),
				nodx.Group(actionNodes...),
			),
		),
	)
}

func executionModal(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
	access rbac.Access,
) component.ModalResult {
	view := executionDetailsViewFromPaginate(execution)

	return component.Modal(component.ModalParams{
		ID:            executionModalID(execution.ID),
		Title:         "Execution details",
		Size:          component.SizeMd,
		HTMXIndicator: executionDetailsLoadingID(execution.ID),
		Content: []nodx.Node{
			renderExecutionDetails(view, access, view.Status == "running"),
		},
	})
}

func executionModalTemplate(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
	access rbac.Access,
) nodx.Node {
	return executionModal(execution, access).HTML
}

func executionActionsButtons(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
	access rbac.Access,
) nodx.Node {
	mo := executionModal(execution, access)

	items := []nodx.Node{
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Eye(),
			component.SpanText(i18n.BtnShowDetails),
		),
	}
	if execution.Status == "failed" && access.CanExecutePbwName(execution.BackupName) {
		items = append(items, retryExecutionButton(execution))
	}

	return nodx.Group(items...)
}
