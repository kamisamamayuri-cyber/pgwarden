package backups

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/backups"
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
		c, http.StatusOK, listBackups(access, formData.Host, pagination, backups),
	)
}

func listBackups(
	access rbac.Access,
	host string,
	pagination paginateutil.PaginateResponse,
	backups []dbgen.BackupsServicePaginateBackupsRow,
) nodx.Node {
	if len(backups) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No backups found",
			Subtitle: "Backups will appear here after they are added",
		})
	}

	yesNoSpan := func(b bool) nodx.Node {
		if b {
			return component.SpanText(i18n.LabelYes)
		}
		return component.SpanText(i18n.LabelNo)
	}

	filterQuery := backupsFilterQuery{Host: host}

	trs := []nodx.Node{}
	for _, backup := range backups {
		menuItems := []nodx.Node{
			component.OptionsDropdownA(
				nodx.Class("btn btn-sm btn-ghost btn-square"),
				nodx.Href(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/executions?backup=%s", backup.ID),
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

		trs = append(trs, nodx.Tr(
			nodx.Td(component.OptionsDropdown(nodx.Group(menuItems...))),
			nodx.Td(
				nodx.Div(
					nodx.Class("flex items-center space-x-2"),
					component.IsActivePing(backup.IsActive),
					component.SpanText(backup.Name),
				),
			),
			nodx.Td(component.SpanText(backup.DatabaseName)),
			nodx.Td(destCell),
			nodx.Td(
				nodx.Class("font-mono"),
				nodx.Div(
					nodx.Class("flex flex-col items-start text-xs"),
					component.SpanText(backup.CronExpression),
					component.SpanText(backup.TimeZone),
				),
			),
			nodx.Td(
				nodx.If(
					backup.RetentionDays == 0,
					lucide.Infinity(),
				),
				nodx.If(
					backup.RetentionDays > 0,
					component.SpanText(fmt.Sprintf("%d d.", backup.RetentionDays)),
				),
			),
			nodx.Td(yesNoSpan(backup.OptDataOnly)),
			nodx.Td(yesNoSpan(backup.OptSchemaOnly)),
			nodx.Td(yesNoSpan(backup.OptClean)),
			nodx.Td(yesNoSpan(backup.OptIfExists)),
			nodx.Td(yesNoSpan(backup.OptCreate)),
			nodx.Td(yesNoSpan(backup.OptNoComments)),
			nodx.Td(component.SpanText(
				backup.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
		))
	}

	if pagination.HasNextPage {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(buildBackupsListURL(filterQuery, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}
