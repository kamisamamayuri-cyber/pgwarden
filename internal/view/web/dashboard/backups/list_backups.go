package backups

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/backups"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/executions"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) listBackupsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var formData struct {
		Host string `query:"host" validate:"omitempty,max=253"`
		Page int    `query:"page" validate:"required,min=1"`
	}
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	access := reqctx.GetCtx(c).Access

	pagination, backups, err := h.servs.BackupsService.PaginateBackups(
		ctx, backups.PaginateBackupsParams{
			HostFilter:  formData.Host,
			Page:        formData.Page,
			Limit:       20,
			NamesFilter: access.NamesFilter(),
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK,
		listBackups(access, h.servs.ExecutionsService, formData.Host, pagination, backups),
	)
}

func listBackups(
	access rbac.Access,
	execs *executions.Service,
	host string,
	pagination paginateutil.PaginateResponse,
	backups []dbgen.BackupsServicePaginateBackupsRow,
) nodx.Node {
	if len(backups) < 1 {
		return component.EmptyResults(component.EmptyResultsParams{
			Title:    "No backups found",
			Subtitle: "Backups will appear here after they are added",
		})
	}

	filterQuery := backupsFilterQuery{Host: host}

	cards := []nodx.Node{}
	for _, backup := range backups {
		menuItems := []nodx.Node{
			component.OptionsDropdownA(
				nodx.Class("btn btn-sm btn-ghost btn-square"),
				nodx.Href(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/jobs?backup=%s", backup.ID),
				)),
				nodx.Target("_blank"),
				lucide.List(),
				component.SpanText(i18n.BtnShowTasks),
			),
		}
		if access.CanExecutePbwName(backup.Name) {
			menuItems = append(menuItems, manualRunbutton(backup.ID))
		}
		if access.CanManageApp() {
			menuItems = append(menuItems,
				editBackupButton(backup),
				duplicateBackupButton(backup.ID),
				deleteBackupButton(backup.ID),
			)
		}

		destCell := nodx.Node(component.PrettyDestinationName(
			backup.IsLocal, backup.DestinationName,
		))
		if !access.CanManageApp() {
			destCell = component.SpanText("S3")
		}

		retention := nodx.Node(lucide.Infinity())
		if backup.RetentionDays > 0 {
			retention = component.SpanText(fmt.Sprintf("%d d.", backup.RetentionDays))
		}

		parallelLabel := "Parallel"
		if backup.ParallelDumpEnabled {
			parallelLabel = fmt.Sprintf("Parallel ×%d", execs.ResolveParallelDumpJobs(backup.ParallelDumpJobs))
		}

		cards = append(cards, component.ItemCard(
			nil,
			[]nodx.Node{
				component.OptionsDropdown(nodx.Group(menuItems...)),
				component.IsActivePing(backup.IsActive),
				nodx.SpanEl(nodx.Class("font-semibold flex-1 truncate"), component.SpanText(backup.Name)),
				component.ToggleBadge("Monthly", backup.MonthlyRetentionEnabled),
				component.ToggleBadge(parallelLabel, backup.ParallelDumpEnabled),
				nodx.If(backup.Tag != "default", component.ToggleBadge(backup.Tag, true)),
			},
			[]nodx.Node{
				component.Stat("Database", component.SpanText(backup.DatabaseName)),
				component.Stat("Destination", destCell),
				component.Stat("Schedule", nodx.SpanEl(
					nodx.Class("font-mono"),
					component.SpanText(backup.CronExpression+" "+backup.TimeZone),
				)),
				component.Stat("Retention", retention),
				component.Stat("Added", component.SpanText(
					backup.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
			},
		))
	}

	if pagination.HasNextPage {
		cards = append(cards, nodx.Div(
			htmx.HxGet(buildBackupsListURL(filterQuery, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(cards)
}
