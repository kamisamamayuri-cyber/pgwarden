package backups

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/staticdata"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	webaccess "github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/access"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	alpine "github.com/nodxdev/nodxgo-alpine"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) createBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	var formData struct {
		DatabaseID                uuid.UUID `form:"database_id" validate:"required,uuid"`
		DestinationID             uuid.UUID `form:"destination_id" validate:"omitempty,uuid"`
		IsLocal                   string    `form:"is_local" validate:"required,oneof=true false"`
		Name                      string    `form:"name" validate:"required"`
		CronExpression            string    `form:"cron_expression" validate:"required"`
		TimeZone                  string    `form:"time_zone" validate:"required"`
		IsActive                  string    `form:"is_active" validate:"required,oneof=true false"`
		DestDir                   string    `form:"dest_dir" validate:"required"`
		RetentionDays             int16     `form:"retention_days"`
		MonthlyRetentionEnabled   string    `form:"monthly_retention_enabled" validate:"required,oneof=true false"`
		OptDataOnly               string    `form:"opt_data_only" validate:"omitempty,oneof=true false"`
		OptSchemaOnly             string    `form:"opt_schema_only" validate:"omitempty,oneof=true false"`
		OptClean                  string    `form:"opt_clean" validate:"required,oneof=true false"`
		OptIfExists               string    `form:"opt_if_exists" validate:"required,oneof=true false"`
		OptCreate                 string    `form:"opt_create" validate:"required,oneof=true false"`
		OptNoComments             string    `form:"opt_no_comments" validate:"required,oneof=true false"`
		OptSerializableDeferrable string    `form:"opt_serializable_deferrable" validate:"omitempty,oneof=true false"`
		ParallelDumpEnabled       string    `form:"parallel_dump_enabled" validate:"required,oneof=true false"`
		Tag                       string    `form:"tag" validate:"omitempty,max=63"`
	}
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	tag := formData.Tag
	if tag == "" {
		tag = "default"
	}

	_, err := h.servs.BackupsService.CreateBackup(
		ctx, dbgen.BackupsServiceCreateBackupParams{
			DatabaseID: formData.DatabaseID,
			DestinationID: uuid.NullUUID{
				Valid: formData.IsLocal == "false", UUID: formData.DestinationID,
			},
			IsLocal:                   formData.IsLocal == "true",
			Name:                      formData.Name,
			CronExpression:            formData.CronExpression,
			TimeZone:                  formData.TimeZone,
			IsActive:                  formData.IsActive == "true",
			DestDir:                   formData.DestDir,
			RetentionDays:             formData.RetentionDays,
			MonthlyRetentionEnabled:   formData.MonthlyRetentionEnabled == "true",
			OptDataOnly:               formData.OptDataOnly == "true",
			OptSchemaOnly:             formData.OptSchemaOnly == "true",
			OptClean:                  formData.OptClean == "true",
			OptIfExists:               formData.OptIfExists == "true",
			OptCreate:                 formData.OptCreate == "true",
			OptNoComments:             formData.OptNoComments == "true",
			OptSerializableDeferrable: formData.OptSerializableDeferrable == "true",
			ParallelDumpEnabled:       formData.ParallelDumpEnabled == "true",
			Tag:                       tag,
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.Redirect(c, pathutil.BuildPath("/dashboard/backups"))
}

func (h *handlers) createBackupFormHandler(c echo.Context) error {
	ctx := c.Request().Context()

	databases, err := h.servs.DatabasesService.GetAllDatabases(ctx)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	destinations, err := h.servs.DestinationsService.GetAllDestinations(ctx)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, createBackupForm(databases, destinations),
	)
}

func createBackupForm(
	databases []dbgen.DatabasesServiceGetAllDatabasesRow,
	destinations []dbgen.DestinationsServiceGetAllDestinationsRow,
) nodx.Node {
	yesNoOptions := func() nodx.Node {
		return nodx.Group(
			nodx.Option(nodx.Value("true"), nodx.Text(i18n.LabelYes)),
			nodx.Option(nodx.Value("false"), nodx.Text(i18n.LabelNo), nodx.Selected("")),
		)
	}

	serverTZ := time.Now().Location().String()

	return nodx.FormEl(
		htmx.HxPost(pathutil.BuildPath("/dashboard/backups")),
		htmx.HxDisabledELT("find button"),
		nodx.Class("space-y-2 text-base"),

		alpine.XData(`{
			is_local: "false",
			parallel_dump: "false",
		}`),

		component.InputControl(component.InputControlParams{
			Name:        "name",
			Label:       i18n.LabelName,
			Placeholder: "My backup",
			Required:    true,
			Type:        component.InputTypeText,
		}),

		component.SelectControl(component.SelectControlParams{
			Name:        "database_id",
			Label:       i18n.LabelDatabase,
			Required:    true,
			Placeholder: "Select database",
			Children: []nodx.Node{
				nodx.Map(
					databases,
					func(db dbgen.DatabasesServiceGetAllDatabasesRow) nodx.Node {
						return nodx.Option(nodx.Value(db.ID.String()), nodx.Text(db.Name))
					},
				),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:     "is_local",
			Label:    "Local backup",
			Required: true,
			Children: []nodx.Node{
				alpine.XModel("is_local"),
				nodx.Option(nodx.Value("true"), nodx.Text(i18n.LabelYes)),
				nodx.Option(nodx.Value("false"), nodx.Text(i18n.LabelNo), nodx.Selected("")),
			},
			HelpButtonChildren: localBackupsHelp(),
		}),

		alpine.Template(
			alpine.XIf("is_local == 'false'"),
			component.SelectControl(component.SelectControlParams{
				Name:        "destination_id",
				Label:       i18n.LabelDestination,
				Required:    true,
				Placeholder: "Select destination",
				Children: []nodx.Node{
					nodx.Map(
						destinations,
						func(dest dbgen.DestinationsServiceGetAllDestinationsRow) nodx.Node {
							return nodx.Option(nodx.Value(dest.ID.String()), nodx.Text(dest.Name))
						},
					),
				},
			}),
		),

		component.InputControl(component.InputControlParams{
			Name:               "cron_expression",
			Label:              "Cron expression",
			Placeholder:        "* * * * *",
			Required:           true,
			Type:               component.InputTypeText,
			HelpText:           "Cron expression for the backup schedule",
			Pattern:            `^\S+\s+\S+\s+\S+\s+\S+\s+\S+$`,
			HelpButtonChildren: cronExpressionHelp(),
		}),

		component.SelectControl(component.SelectControlParams{
			Name:        "time_zone",
			Label:       "Timezone",
			Required:    true,
			Placeholder: "Select timezone",
			Children: []nodx.Node{
				nodx.Map(
					staticdata.Timezones,
					func(tz staticdata.Timezone) nodx.Node {
						var selected nodx.Node
						if tz.TzCode == serverTZ {
							selected = nodx.Selected("")
						}

						return nodx.Option(nodx.Value(tz.TzCode), nodx.Text(tz.Label), selected)
					},
				),
			},
			HelpButtonChildren: timezoneFilenamesHelp(),
		}),

		component.InputControl(component.InputControlParams{
			Name:               "dest_dir",
			Label:              "Destination directory",
			Placeholder:        "/path/to/backup",
			Required:           true,
			Type:               component.InputTypeText,
			HelpText:           "Relative to the destination base directory",
			Pattern:            `^\/\S*[^\/]$`,
			HelpButtonChildren: destinationDirectoryHelp(),
		}),

		component.InputControl(component.InputControlParams{
			Name:               "retention_days",
			Label:              "Retention (days)",
			Placeholder:        "30",
			Required:           true,
			Type:               component.InputTypeNumber,
			Pattern:            "[0-9]+",
			HelpButtonChildren: retentionDaysHelp(),
			Children: []nodx.Node{
				nodx.Min("0"),
				nodx.Max("36500"),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:               "monthly_retention_enabled",
			Label:              "Keep monthly backups",
			Required:           true,
			HelpButtonChildren: monthlyRetentionHelp(),
			Children: []nodx.Node{
				yesNoOptions(),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:               "parallel_dump_enabled",
			Label:              "Parallel dump",
			Required:           true,
			HelpButtonChildren: parallelDumpHelp(),
			Children: []nodx.Node{
				alpine.XModel("parallel_dump"),
				yesNoOptions(),
			},
		}),

		component.InputControl(component.InputControlParams{
			Name:        "tag",
			Label:       "Worker tag",
			Placeholder: "default",
			Type:        component.InputTypeText,
			HelpText:    "Only workers configured with a matching PBW_WORKER_TAGS entry claim this backup. Empty = \"default\"",
		}),

		component.SelectControl(component.SelectControlParams{
			Name:     "is_active",
			Label:    "Activate backup",
			Required: true,
			Children: []nodx.Node{
				nodx.Option(nodx.Value("true"), nodx.Text(i18n.LabelYes)),
				nodx.Option(nodx.Value("false"), nodx.Text(i18n.LabelNo)),
			},
		}),

		nodx.Div(
			nodx.Class("pt-4"),
			nodx.Div(
				nodx.Class("flex justify-start items-center space-x-1"),
				component.H2Text("Options"),
				component.HelpButtonModal(component.HelpButtonModalParams{
					ModalTitle: "pg_dump options",
					Children:   pgDumpOptionsHelp(),
				}),
			),

			nodx.Div(
				nodx.Class("mt-2 grid grid-cols-2 gap-2"),

				component.SelectControl(component.SelectControlParams{
					Name:  "opt_data_only",
					Label: "--data-only",
					Children: []nodx.Node{
						parallelDumpDisables(),
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:  "opt_schema_only",
					Label: "--schema-only",
					Children: []nodx.Node{
						parallelDumpDisables(),
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_clean",
					Label:    "--clean",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_if_exists",
					Label:    "--if-exists",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_create",
					Label:    "--create",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_no_comments",
					Label:    "--no-comments",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:  "opt_serializable_deferrable",
					Label: "--serializable-deferrable",
					Children: []nodx.Node{
						parallelDumpDisables(),
						yesNoOptions(),
					},
					HelpButtonChildren: serializableDeferrableHelp(),
				}),
			),
		),

		nodx.Div(
			nodx.Class("flex justify-end items-center space-x-2 pt-2"),
			component.HxLoadingMd(),
			nodx.Button(
				nodx.Class("btn btn-primary"),
				nodx.Type("submit"),
				component.SpanText("Create backup"),
				lucide.Save(),
			),
		),
	)
}

func createBackupButton() nodx.Node {
	mo := component.Modal(component.ModalParams{
		Size:  component.SizeLg,
		Title: "Create backup",
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath("/dashboard/backups/create-form")),
				htmx.HxSwap("outerHTML"),
				htmx.HxTrigger("intersect once"),
				nodx.Class("p-10 flex justify-center"),
				component.HxLoadingMd(),
			),
		},
	})

	button := nodx.Button(
		mo.OpenerAttr,
		nodx.Class("btn btn-primary"),
		component.SpanText("Create backup"),
		lucide.Plus(),
	)

	return nodx.Div(
		nodx.Class("inline-block"),
		mo.HTML,
		button,
	)
}
