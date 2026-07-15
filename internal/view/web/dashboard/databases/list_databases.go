package databases

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/databases"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) listDatabasesHandler(c echo.Context) error {
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

	pagination, databases, err := h.servs.DatabasesService.PaginateDatabases(
		ctx, databases.PaginateDatabasesParams{
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
		c, http.StatusOK, listDatabases(access, formData.Host, pagination, databases),
	)
}

func listDatabases(
	access rbac.Access,
	host string,
	pagination paginateutil.PaginateResponse,
	databases []dbgen.DatabasesServicePaginateDatabasesRow,
) nodx.Node {
	if len(databases) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No databases found",
			Subtitle: "Databases will appear here after they are added",
		})
	}

	filterQuery := databasesFilterQuery{Host: host}

	trs := []nodx.Node{}
	for _, database := range databases {
		menuItems := []nodx.Node{
			component.OptionsDropdownA(
				nodx.Href(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/executions?database=%s", database.ID),
				)),
				nodx.Target("_blank"),
				lucide.List(),
				component.SpanText(i18n.BtnShowTasks),
			),
		}
		if access.CanManageApp() {
			menuItems = append(menuItems,
				editDatabaseButton(database),
				component.OptionsDropdownButton(
					htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/databases/%s/test", database.ID))),
					htmx.HxDisabledELT("this"),
					lucide.DatabaseZap(),
					component.SpanText(i18n.BtnTestConnection),
				),
				deleteDatabaseButton(database.ID),
			)
		}

		connCell := []nodx.Node{component.SpanText("****************")}
		if access.CanSeeConnectionSecrets() {
			connCell = []nodx.Node{
				component.CopyButtonSm(database.DecryptedConnectionString),
				component.SpanText("****************"),
			}
		}

		trs = append(trs, nodx.Tr(
			nodx.Td(component.OptionsDropdown(
				nodx.Div(
					nodx.Class("flex flex-col space-y-1"),
					nodx.Group(menuItems...),
				),
			)),
			nodx.Td(
				nodx.Div(
					nodx.Class("flex items-center space-x-2"),
					component.HealthStatusPing(
						database.TestOk, database.TestError, database.LastTestAt,
					),
					component.SpanText(database.Name),
				),
			),
			nodx.Td(component.SpanText("PostgreSQL "+database.PgVersion)),
			nodx.Td(
				nodx.Class("space-x-1"),
				nodx.Group(connCell...),
			),
			nodx.Td(component.SpanText(
				database.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
		))
	}

	if pagination.HasNextPage {
		trs = append(trs, nodx.Tr(
			htmx.HxGet(buildDatabasesListURL(filterQuery, pagination.NextPage)),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(trs)
}
