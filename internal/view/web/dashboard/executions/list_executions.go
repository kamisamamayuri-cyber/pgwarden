package executions

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type listExecsQueryData struct {
	Database    uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup      uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	Status      string    `query:"status" validate:"omitempty,oneof=queued running success failed deleted"`
	Host        string    `query:"host" validate:"omitempty,max=253"`
	Page        int       `query:"page" validate:"required,min=1"`
}

func (h *handlers) listExecutionsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var queryData listExecsQueryData
	if err := c.Bind(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access

	pagination, executions, err := h.servs.ExecutionsService.PaginateExecutions(
		ctx, executions.PaginateExecutionsParams{
			DatabaseFilter: uuid.NullUUID{
				UUID: queryData.Database, Valid: queryData.Database != uuid.Nil,
			},
			DestinationFilter: uuid.NullUUID{
				UUID: queryData.Destination, Valid: queryData.Destination != uuid.Nil,
			},
			BackupFilter: uuid.NullUUID{
				UUID: queryData.Backup, Valid: queryData.Backup != uuid.Nil,
			},
			StatusFilter: queryData.Status,
			HostFilter:   queryData.Host,
			Page:         queryData.Page,
			Limit:        20,
			NamesFilter:  access.NamesFilter(),
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, listExecutions(access, queryData, pagination, executions),
	)
}

func listExecutions(
	access rbac.Access,
	queryData listExecsQueryData,
	pagination paginateutil.PaginateResponse,
	executions []dbgen.ExecutionsServicePaginateExecutionsRow,
) nodx.Node {
	if len(executions) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No executions found",
			Subtitle: "Executions will appear here after backups run",
		})
	}

	filterQuery := execsFilterQuery{
		Database:    queryData.Database,
		Destination: queryData.Destination,
		Backup:      queryData.Backup,
		Status:      queryData.Status,
		Host:        queryData.Host,
	}

	trs := []nodx.Node{}
	for _, execution := range executions {
		destCell := nodx.Node(component.PrettyDestinationName(
			execution.BackupIsLocal, execution.DestinationName,
		))
		if !access.CanManageApp() {
			destCell = component.SpanText("S3")
		}

		trs = append(trs, nodx.Tr(
			nodx.Td(component.OptionsDropdown(
				showExecutionButton(execution, access),
				restoreExecutionButton(execution, access),
			)),
			nodx.Td(component.StatusBadge(execution.Status)),
			nodx.Td(component.SpanText(execution.BackupName)),
			nodx.Td(component.SpanText(execution.DatabaseName)),
			nodx.Td(destCell),
			nodx.Td(component.SpanText(
				execution.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
			nodx.Td(
				nodx.If(
					execution.FinishedAt.Valid,
					component.SpanText(
						execution.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
					),
				),
			),
			nodx.Td(
				nodx.If(
					execution.FinishedAt.Valid,
					component.SpanText(
						execution.FinishedAt.Time.Sub(execution.StartedAt).String(),
					),
				),
			),
			nodx.Td(
				nodx.If(
					execution.FileSize.Valid,
					component.PrettyFileSize(execution.FileSize),
				),
			),
		))
	}

	if pagination.HasNextPage {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(buildExecutionsListURL(filterQuery, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}
