package restorations

import (
	"database/sql"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
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

type listResQueryData struct {
	Execution uuid.UUID `query:"execution" validate:"omitempty,uuid"`
	Database  uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Status    string    `query:"status" validate:"omitempty,oneof=queued running success failed"`
	Page      int       `query:"page" validate:"required,min=1"`
}

func (h *handlers) listRestorationsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var queryData listResQueryData
	if err := c.Bind(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access

	filterQuery := resFilterQuery{
		Execution: queryData.Execution,
		Database:  queryData.Database,
		Status:    queryData.Status,
	}

	pagination, restorationsList, err := h.servs.RestorationsService.PaginateRestorations(
		ctx, restorations.PaginateRestorationsParams{
			ExecutionFilter: uuid.NullUUID{
				UUID: queryData.Execution, Valid: queryData.Execution != uuid.Nil,
			},
			DatabaseFilter: uuid.NullUUID{
				UUID: queryData.Database, Valid: queryData.Database != uuid.Nil,
			},
			StatusFilter: sql.NullString{
				String: queryData.Status, Valid: queryData.Status != "",
			},
			Page:        queryData.Page,
			Limit:       20,
			NamesFilter: access.NamesFilter(),
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, listRestorations(filterQuery, pagination, restorationsList),
	)
}

func listRestorations(
	filterQuery resFilterQuery,
	pagination paginateutil.PaginateResponse,
	restorationsList []dbgen.RestorationsServicePaginateRestorationsRow,
) nodx.Node {
	if len(restorationsList) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No restorations found",
			Subtitle: "Restorations will appear here after a restore is started",
		})
	}

	trs := []nodx.Node{}
	for _, restoration := range restorationsList {
		trs = append(trs, nodx.Tr(
			nodx.Td(
				showRestorationButton(restoration),
			),
			nodx.Td(component.StatusBadge(restoration.Status)),
			nodx.Td(component.SpanText(restoration.BackupName)),
			nodx.Td(component.SpanText(restorationTargetDatabase(restoration))),
			nodx.Td(component.SpanText(restoration.ExecutionID.String())),
			nodx.Td(component.SpanText(
				restoration.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
			nodx.Td(
				nodx.If(
					restoration.FinishedAt.Valid,
					component.SpanText(
						restoration.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
					),
				),
			),
			nodx.Td(
				component.SpanText(
					restorations.RestorationDuration(
						restoration.StartedAt, restoration.FinishedAt,
					),
				),
			),
		))
	}

	if pagination.HasNextPage {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(buildRestorationsListURL(filterQuery, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}
