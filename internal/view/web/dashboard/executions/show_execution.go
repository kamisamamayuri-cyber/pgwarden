package executions

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
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

	if isLocal {
		return c.Attachment(link, filepath.Base(link))
	}

	return c.Redirect(http.StatusFound, link)
}

func showExecutionButton(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
	access rbac.Access,
) nodx.Node {
	destCell := nodx.Node(component.PrettyDestinationName(
		execution.BackupIsLocal, execution.DestinationName,
	))
	if !access.CanManageApp() {
		destCell = component.SpanText("S3")
	}

	actionNodes := []nodx.Node{}
	if execution.Status == "success" {
		if access.CanManageApp() {
			actionNodes = append(actionNodes, deleteExecutionButton(execution.ID))
		}
		if access.CanViewPbwName(execution.DatabaseName) {
			actionNodes = append(actionNodes, nodx.A(
				nodx.Href(pathutil.BuildPath(fmt.Sprintf("/dashboard/executions/%s/download", execution.ID))),
				nodx.Target("_blank"),
				nodx.Class("btn btn-primary"),
				component.SpanText(i18n.BtnDownload),
				lucide.Download(),
			))
		}
	}

	mo := component.Modal(component.ModalParams{
		Title: "Execution details",
		Size:  component.SizeMd,
		Content: []nodx.Node{
			nodx.Div(
				nodx.Class("overflow-x-auto"),
				nodx.Table(
					nodx.Class("table [&_th]:text-nowrap"),
					nodx.Tr(
						nodx.Th(component.SpanText("ID")),
						nodx.Td(component.SpanText(execution.ID.String())),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelStatus)),
						nodx.Td(component.StatusBadge(execution.Status)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelDatabase)),
						nodx.Td(component.SpanText(execution.DatabaseName)),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelDestination)),
						nodx.Td(destCell),
					),
					nodx.If(
						execution.Message.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText(i18n.LabelMessage)),
							nodx.Td(
								nodx.Class("break-all"),
								component.SpanText(execution.Message.String),
							),
						),
					),
					nodx.Tr(
						nodx.Th(component.SpanText(i18n.LabelStartedAt)),
						nodx.Td(component.SpanText(
							execution.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
						)),
					),
					nodx.If(
						execution.FinishedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText(i18n.LabelFinishedAt)),
							nodx.Td(component.SpanText(
								execution.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
							)),
						),
					),
					nodx.If(
						execution.FinishedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText(i18n.LabelDuration)),
							nodx.Td(component.SpanText(
								execution.FinishedAt.Time.Sub(execution.StartedAt).String(),
							)),
						),
					),
					nodx.If(
						execution.DeletedAt.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText("Deleted at")),
							nodx.Td(component.SpanText(
								execution.DeletedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
							)),
						),
					),
					nodx.If(
						execution.FileSize.Valid,
						nodx.Tr(
							nodx.Th(component.SpanText(i18n.LabelFileSize)),
							nodx.Td(component.PrettyFileSize(execution.FileSize)),
						),
					),
				),
				nodx.If(
					len(actionNodes) > 0,
					nodx.Div(
						nodx.Class("flex justify-end items-center space-x-2"),
						nodx.Group(actionNodes...),
					),
				),
			),
		},
	})

	return component.RenderableGroup([]nodx.Node{
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Eye(),
			component.SpanText(i18n.BtnShowDetails),
		),
	})
}
