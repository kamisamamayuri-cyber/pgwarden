package databases

import (
	"database/sql"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) editDatabaseHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	databaseID, err := uuid.Parse(c.Param("databaseID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var formData createDatabaseDTO
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	_, err = h.servs.DatabasesService.UpdateDatabase(
		ctx, dbgen.DatabasesServiceUpdateDatabaseParams{
			ID:               databaseID,
			Name:             sql.NullString{String: formData.Name, Valid: true},
			PgVersion:        sql.NullString{String: formData.Version, Valid: true},
			ConnectionString: sql.NullString{String: formData.ConnectionString, Valid: true},
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.AlertWithRefresh(c, "Database updated")
}

func editDatabaseButton(
	database dbgen.DatabasesServicePaginateDatabasesRow,
) nodx.Node {
	idPref := "edit-database-" + database.ID.String()
	formID := idPref + "-form"
	btnClass := idPref + "-btn"
	loadingID := idPref + "-loading"

	htmxAttributes := func(url string) nodx.Node {
		return nodx.Group(
			htmx.HxPost(pathutil.BuildPath(url)),
			htmx.HxInclude("#"+formID),
			htmx.HxDisabledELT("."+btnClass),
			htmx.HxIndicator("#"+loadingID),
			htmx.HxValidate("true"),
		)
	}

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeMd,
		Title: "Edit database",
		Content: []nodx.Node{
			nodx.FormEl(
				nodx.Id(formID),
				nodx.Class("space-y-2"),

				component.InputControl(component.InputControlParams{
					Name:        "name",
					Label:       i18n.LabelName,
					Placeholder: "My database",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "A name for easy identification of the database",
					Children: []nodx.Node{
						nodx.Value(database.Name),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "version",
					Label:    i18n.LabelVersion,
					Required: true,
					HelpText: "PostgreSQL version",
					Children: []nodx.Node{
						component.PGVersionSelectOptions(sql.NullString{
							Valid:  true,
							String: database.PgVersion,
						}),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "connection_string",
					Label:       i18n.LabelConnectionString,
					Placeholder: "postgresql://user:password@localhost:5432/mydb",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "Must be a valid PostgreSQL connection string with database name. Stored with PGP encryption.",
					Children: []nodx.Node{
						nodx.Value(database.DecryptedConnectionString),
					},
				}),
			),

			nodx.Div(
				nodx.Class("flex justify-between items-center pt-4"),
				nodx.Div(
					nodx.Button(
						htmxAttributes("/dashboard/databases/test"),
						nodx.ClassMap{
							btnClass:                      true,
							"btn btn-neutral btn-outline": true,
						},
						nodx.Type("button"),
						component.SpanText(i18n.BtnTestConnection),
						lucide.DatabaseZap(),
					),
				),
				nodx.Div(
					nodx.Class("flex justify-end items-center space-x-2"),
					component.HxLoadingMd(loadingID),
					nodx.Button(
						htmxAttributes("/dashboard/databases/"+database.ID.String()+"/edit"),
						nodx.ClassMap{
							btnClass:          true,
							"btn btn-primary": true,
						},
						nodx.Type("button"),
						component.SpanText(i18n.BtnSave),
						lucide.Save(),
					),
				),
			),
		},
	})

	return component.RenderableGroup([]nodx.Node{
		mo.HTML,
		component.OptionsDropdownButton(
			mo.OpenerAttr,
			lucide.Pencil(),
			component.SpanText("Edit database"),
		),
	})
}
