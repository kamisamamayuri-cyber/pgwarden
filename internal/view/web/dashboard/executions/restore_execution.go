package executions

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/rbac"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	alpine "github.com/nodxdev/nodxgo-alpine"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) restoreExecutionHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var formData struct {
		ExecutionID uuid.UUID `form:"execution_id" validate:"required,uuid"`
		DatabaseID  uuid.UUID `form:"database_id" validate:"omitempty,uuid"`
		ConnString  string    `form:"conn_string" validate:"omitempty"`
	}
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	if formData.DatabaseID == uuid.Nil && formData.ConnString == "" {
		return respondhtmx.ToastError(
			c, "Specify a database or connection string",
		)
	}

	if formData.DatabaseID != uuid.Nil && formData.ConnString != "" {
		return respondhtmx.ToastError(
			c, "Cannot specify both a database and a connection string",
		)
	}

	execution, err := h.servs.ExecutionsService.GetExecution(
		ctx, formData.ExecutionID,
	)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, execution.BackupID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanExecutePbwName(backup.Name) {
		return webaccess.ForbiddenHTMX(c)
	}

	if formData.ConnString != "" {
		err := h.servs.DatabasesService.TestDatabase(
			ctx, execution.DatabasePgVersion, formData.ConnString,
		)
		if err != nil {
			return respondhtmx.ToastError(c, err.Error())
		}
	}

	_, err = h.servs.RestorationsService.EnqueueRestoration(
		ctx, restorations.EnqueueRestorationParams{
			ExecutionID: formData.ExecutionID,
			DatabaseID: uuid.NullUUID{
				Valid: formData.DatabaseID != uuid.Nil,
				UUID:  formData.DatabaseID,
			},
			ConnString: formData.ConnString,
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.ToastSuccess(
		c, "Restore queued, see details on the restorations page",
	)
}

func (h *handlers) restoreExecutionFormHandler(c echo.Context) error {
	ctx := c.Request().Context()

	executionID, err := uuid.Parse(c.Param("executionID"))
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	execution, err := h.servs.ExecutionsService.GetExecution(ctx, executionID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(ctx, execution.BackupID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	access := reqctx.GetCtx(c).Access
	if !access.CanExecutePbwName(backup.Name) {
		return webaccess.ForbiddenHTMX(c)
	}

	databases, err := h.servs.DatabasesService.GetAllDatabases(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, restoreExecutionForm(
		execution, databases,
	))
}

func restoreExecutionForm(
	execution dbgen.ExecutionsServiceGetExecutionRow,
	databases []dbgen.DatabasesServiceGetAllDatabasesRow,
) nodx.Node {
	return nodx.FormEl(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/executions/%s/restore", execution.ID))),
		htmx.HxConfirm("Restore this backup?"),
		htmx.HxDisabledELT("find button"),

		alpine.XData(`{ backup_to: "database" }`),

		nodx.Input(
			nodx.Type("hidden"),
			nodx.Name("execution_id"),
			nodx.Value(execution.ID.String()),
		),

		nodx.Div(
			nodx.Class("space-y-2 text-base"),

			component.SelectControl(component.SelectControlParams{
				Name:     "backup_to",
				Label:    "Restore to",
				Required: true,
				HelpText: "Restore to an existing database or to another via connection string",
				Children: []nodx.Node{
					alpine.XModel("backup_to"),
					nodx.Option(
						nodx.Value("database"),
						nodx.Text("Existing database"),
						nodx.Selected(""),
					),
					nodx.Option(
						nodx.Value("conn_string"),
						nodx.Text("Other database"),
					),
				},
			}),

			alpine.Template(
				alpine.XIf("backup_to === 'database'"),
				component.SelectControl(component.SelectControlParams{
					Name:        "database_id",
					Label:       i18n.LabelDatabase,
					Placeholder: "Select database",
					Required:    true,
					Children: []nodx.Node{
						nodx.Map(
							databases,
							func(db dbgen.DatabasesServiceGetAllDatabasesRow) nodx.Node {
								return nodx.Option(
									nodx.Value(db.ID.String()),
									nodx.Text(db.Name),
									nodx.If(
										db.ID == execution.DatabaseID,
										nodx.Selected(""),
									),
								)
							},
						),
					},
				}),
			),

			alpine.Template(
				alpine.XIf("backup_to === 'conn_string'"),
				component.InputControl(component.InputControlParams{
					Name:        "conn_string",
					Label:       i18n.LabelConnectionString,
					Placeholder: "postgresql://user:password@localhost:5432/mydb",
					Type:        component.InputTypeText,
					Required:    true,
				}),
			),

			nodx.Div(
				nodx.Class("pt-2"),
				nodx.Div(
					nodx.Role("alert"),
					nodx.Class("alert alert-warning"),
					lucide.TriangleAlert(),
					nodx.Div(
						nodx.P(
							component.BText(fmt.Sprintf(
								"Restore uses psql v%s", execution.DatabasePgVersion,
							)),
						),
						component.PText(`
							Make sure the target database is compatible with this version of psql
							and that the correct database is selected for restore.
						`),
					),
				),
			),

			nodx.Div(
				nodx.Class("flex justify-end items-center space-x-2 pt-2"),
				component.HxLoadingMd(),
				nodx.Button(
					nodx.Class("btn btn-primary"),
					nodx.Type("submit"),
					component.SpanText("Start restore"),
					lucide.Zap(),
				),
			),
		),
	)
}

func restoreExecutionButton(
	execution dbgen.ExecutionsServicePaginateExecutionsRow,
	access rbac.Access,
) nodx.Node {
	if execution.Status != "success" || !execution.Path.Valid {
		return nil
	}
	if !access.CanExecutePbwName(execution.BackupName) {
		return nil
	}

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeMd,
		Title: "Restore from execution",
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath(fmt.Sprintf("/dashboard/executions/%s/restore-form", execution.ID))),
				htmx.HxSwap("outerHTML"),
				htmx.HxTrigger("intersect once"),
				nodx.Class("p-10 flex justify-center"),
				component.HxLoadingMd(),
			),
		},
	})

	return component.RenderableGroup([]nodx.Node{
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.ArchiveRestore(),
			component.SpanText("Restore"),
		),
	})
}
