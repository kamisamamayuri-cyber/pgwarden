package executions

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/jobs"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	restorationsService "github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/dashboard/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type listExecsQueryData struct {
	Database    uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup      uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	Status      string    `query:"status" validate:"omitempty,oneof=queued running success failed deleted"`
	Type        string    `query:"type" validate:"omitempty,oneof=backup restore fix_owner"`
	Host        string    `query:"host" validate:"omitempty,max=253"`
	Page        int       `query:"page" validate:"required,min=1"`
}

const executionsPageSize = 20
const executionsPollMaxRows = 400

type jobsPage struct {
	pagination      paginateutil.PaginateResponse
	order           []jobs.JobRef
	backupByID      map[uuid.UUID]dbgen.ExecutionsServicePaginateExecutionsRow
	restorationByID map[uuid.UUID]dbgen.RestorationsServicePaginateRestorationsRow
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
	isPoll := c.QueryParam("poll") == "1"

	limit := executionsPageSize
	if isPoll {
		rows, _ := strconv.Atoi(c.QueryParam("rows"))
		limit = pollLimit(rows, executionsPageSize, executionsPollMaxRows)
	}

	pagination, refs, err := h.servs.JobsService.PaginateJobs(
		ctx, jobs.PaginateJobsParams{
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
			KindFilter:   queryData.Type,
			Page:         queryData.Page,
			Limit:        limit,
			NamesFilter:  access.NamesFilter(),
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	page, err := h.loadJobsPage(ctx, pagination, refs)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if queryData.Page > 1 {
		return echoutil.RenderNodx(
			c, http.StatusOK,
			nodx.Group(
				buildJobCards(access, queryData, page),
				jobModalsOOB(page, access),
			),
		)
	}

	active, err := h.servs.JobsService.HasActiveJobs(ctx)
	if err != nil {
		active = false
	}

	if isPoll {
		page.pagination.NextPage = limit/executionsPageSize + 1
		status := http.StatusOK
		if !active {
			status = htmxStopPollingStatus
		}
		return echoutil.RenderNodx(
			c, status, buildJobCards(access, queryData, page),
		)
	}

	return echoutil.RenderNodx(
		c, http.StatusOK,
		nodx.Group(
			renderJobsList(access, queryData, page, active),
			jobModalsOOB(page, access),
		),
	)
}

func (h *handlers) loadJobsPage(
	ctx context.Context,
	pagination paginateutil.PaginateResponse,
	refs []jobs.JobRef,
) (jobsPage, error) {
	var backupIDs, restorationIDs []uuid.UUID
	for _, r := range refs {
		if r.Kind == jobs.KindBackup {
			backupIDs = append(backupIDs, r.ID)
		} else {
			restorationIDs = append(restorationIDs, r.ID)
		}
	}

	backupRows, err := h.servs.ExecutionsService.GetExecutionsByIDs(ctx, backupIDs)
	if err != nil {
		return jobsPage{}, err
	}
	restorationRows, err := h.servs.RestorationsService.GetRestorationsByIDs(ctx, restorationIDs)
	if err != nil {
		return jobsPage{}, err
	}

	backupByID := make(map[uuid.UUID]dbgen.ExecutionsServicePaginateExecutionsRow, len(backupRows))
	for _, row := range backupRows {
		backupByID[row.ID] = row
	}
	restorationByID := make(map[uuid.UUID]dbgen.RestorationsServicePaginateRestorationsRow, len(restorationRows))
	for _, row := range restorationRows {
		restorationByID[row.ID] = row
	}

	return jobsPage{
		pagination:      pagination,
		order:           refs,
		backupByID:      backupByID,
		restorationByID: restorationByID,
	}, nil
}

const htmxStopPollingStatus = 286

func pollLimit(rows, pageSize, maxRows int) int {
	if rows < pageSize {
		return pageSize
	}
	if rows > maxRows {
		rows = maxRows
	}
	return (rows + pageSize - 1) / pageSize * pageSize
}

const executionsTbodyID = "executions-list"
const executionModalsContainerID = "execution-modals"

func renderJobsList(
	access rbac.Access,
	queryData listExecsQueryData,
	page jobsPage,
	poll bool,
) nodx.Node {
	filterQuery := execsFilterQuery{
		Database:    queryData.Database,
		Destination: queryData.Destination,
		Backup:      queryData.Backup,
		Status:      queryData.Status,
		Type:        queryData.Type,
		Host:        queryData.Host,
	}

	attrs := []nodx.Node{nodx.Id(executionsTbodyID)}
	if poll {
		attrs = append(attrs,
			htmx.HxGet(buildExecutionsListPollURL(filterQuery, 1)),
			htmx.HxTrigger("every 5s"),
			htmx.HxSwap("innerHTML"),
			nodx.Attr("hx-vals",
				"js:{rows: document.querySelectorAll('#"+executionsTbodyID+" > div[data-row]').length}"),
		)
	}
	return nodx.Div(append(attrs, buildJobCards(access, queryData, page))...)
}

func jobModalsOOB(page jobsPage, access rbac.Access) nodx.Node {
	if len(page.order) < 1 {
		return nil
	}

	modals := make([]nodx.Node, 0, len(page.order))
	for _, ref := range page.order {
		if ref.Kind == jobs.KindBackup {
			if row, ok := page.backupByID[ref.ID]; ok {
				modals = append(modals, executionModalTemplate(row, access))
			}
			continue
		}
		if row, ok := page.restorationByID[ref.ID]; ok {
			modals = append(modals, restorations.ModalTemplate(row))
		}
	}

	return nodx.Group(
		nodx.Div(
			nodx.Id(executionModalsContainerID),
			nodx.Attr("hx-swap-oob", "beforeend"),
			nodx.Group(modals...),
		),
	)
}

func executionDurationVisible(status string, finishedAt sql.NullTime) bool {
	if finishedAt.Valid {
		return true
	}
	return status == "queued" || status == "running"
}

func executionDuration(startedAt time.Time, finishedAt sql.NullTime) string {
	end := time.Now()
	if finishedAt.Valid {
		end = finishedAt.Time
	}
	if !end.After(startedAt) {
		return "0s"
	}
	return end.Sub(startedAt).Round(time.Second).String()
}

func jobCard(actions, statusBadge, kindBadge, title nodx.Node, stats []nodx.Node) nodx.Node {
	return component.ItemCard(
		[]nodx.Node{nodx.Data("row", "1")},
		[]nodx.Node{
			actions,
			statusBadge,
			kindBadge,
			nodx.SpanEl(nodx.Class("font-semibold flex-1 truncate"), title),
		},
		stats,
	)
}

func buildJobCards(
	access rbac.Access,
	queryData listExecsQueryData,
	page jobsPage,
) nodx.Node {
	if len(page.order) < 1 {
		return component.EmptyResults(component.EmptyResultsParams{
			Title:    "No jobs found",
			Subtitle: "Jobs will appear here after a backup runs, a restore is started, or a Fix owner action is started",
		})
	}

	filterQuery := execsFilterQuery{
		Database:    queryData.Database,
		Destination: queryData.Destination,
		Backup:      queryData.Backup,
		Status:      queryData.Status,
		Type:        queryData.Type,
		Host:        queryData.Host,
	}

	cards := []nodx.Node{}
	for _, ref := range page.order {
		if ref.Kind != jobs.KindBackup {
			row, ok := page.restorationByID[ref.ID]
			if !ok {
				continue
			}

			title := component.SpanText(restorations.TargetDatabase(row))
			if ref.Kind == jobs.KindRestore {
				title = component.SpanText(fmt.Sprintf(
					"%s → %s", row.BackupName, restorations.TargetDatabase(row),
				))
			}

			restoreStats := []nodx.Node{
				component.Stat("Started", component.SpanText(
					row.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
				component.Stat("Duration", component.SpanText(
					restorationsService.RestorationDuration(row.StartedAt, row.FinishedAt),
				)),
				component.Stat("Target", component.SpanText(restorations.TargetDatabase(row))),
			}
			if row.Tag != "default" {
				restoreStats = append(restoreStats, component.Stat("Tag", component.SpanText(row.Tag)))
			}

			cards = append(cards, jobCard(
				component.OptionsDropdown(restorations.ShowDetailsMenuItem(row)),
				component.StatusBadge(row.Status),
				component.JobKindBadge(ref.Kind),
				title,
				restoreStats,
			))
			continue
		}

		row, ok := page.backupByID[ref.ID]
		if !ok {
			continue
		}

		destCell := nodx.Node(component.PrettyDestinationName(
			row.BackupIsLocal, row.DestinationName,
		))
		if !access.CanManageApp() {
			destCell = component.SpanText("S3")
		}

		stats := []nodx.Node{
			component.Stat("Started", component.SpanText(
				row.StartedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
			component.Stat("Database", component.SpanText(row.DatabaseName)),
			component.Stat("Destination", destCell),
		}
		if executionDurationVisible(row.Status, row.FinishedAt) {
			stats = append(stats, component.Stat("Duration", component.SpanText(
				executionDuration(row.StartedAt, row.FinishedAt),
			)))
		}
		if row.FileSize.Valid {
			stats = append(stats, component.Stat("File size", component.PrettyFileSize(row.FileSize)))
		}
		if row.BackupTag != "default" {
			stats = append(stats, component.Stat("Tag", component.SpanText(row.BackupTag)))
		}

		cards = append(cards, jobCard(
			component.OptionsDropdown(executionActionsButtons(row, access)),
			component.StatusBadge(row.Status),
			component.JobKindBadge(jobs.KindBackup),
			component.SpanText(row.BackupName),
			stats,
		))
	}

	if page.pagination.HasNextPage {
		cards = append(cards, nodx.Div(
			htmx.HxGet(buildExecutionsListURL(filterQuery, page.pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(cards)
}
