package backups

import (
	"database/sql"
	"fmt"
	"net/http"

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

func (h *handlers) editBackupHandler(c echo.Context) error {
	ctx := c.Request().Context()

	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var formData struct {
		Name                      string `form:"name" validate:"required"`
		CronExpression            string `form:"cron_expression" validate:"required"`
		TimeZone                  string `form:"time_zone" validate:"required"`
		IsActive                  string `form:"is_active" validate:"required,oneof=true false"`
		DestDir                   string `form:"dest_dir" validate:"required"`
		RetentionDays             int16  `form:"retention_days"`
		MonthlyRetentionEnabled   string `form:"monthly_retention_enabled" validate:"required,oneof=true false"`
		OptDataOnly               string `form:"opt_data_only" validate:"omitempty,oneof=true false"`
		OptSchemaOnly             string `form:"opt_schema_only" validate:"omitempty,oneof=true false"`
		OptClean                  string `form:"opt_clean" validate:"required,oneof=true false"`
		OptIfExists               string `form:"opt_if_exists" validate:"required,oneof=true false"`
		OptCreate                 string `form:"opt_create" validate:"required,oneof=true false"`
		OptNoComments             string `form:"opt_no_comments" validate:"required,oneof=true false"`
		OptSerializableDeferrable string `form:"opt_serializable_deferrable" validate:"omitempty,oneof=true false"`
		ParallelDumpEnabled       string `form:"parallel_dump_enabled" validate:"required,oneof=true false"`
		Tag                       string `form:"tag" validate:"omitempty,max=63"`
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

	_, err = h.servs.BackupsService.UpdateBackup(
		ctx, dbgen.BackupsServiceUpdateBackupParams{
			ID:                        backupID,
			Name:                      sql.NullString{String: formData.Name, Valid: true},
			CronExpression:            sql.NullString{String: formData.CronExpression, Valid: true},
			TimeZone:                  sql.NullString{String: formData.TimeZone, Valid: true},
			IsActive:                  sql.NullBool{Bool: formData.IsActive == "true", Valid: true},
			DestDir:                   sql.NullString{String: formData.DestDir, Valid: true},
			RetentionDays:             sql.NullInt16{Int16: formData.RetentionDays, Valid: true},
			MonthlyRetentionEnabled:   sql.NullBool{Bool: formData.MonthlyRetentionEnabled == "true", Valid: true},
			OptDataOnly:               sql.NullBool{Bool: formData.OptDataOnly == "true", Valid: true},
			OptSchemaOnly:             sql.NullBool{Bool: formData.OptSchemaOnly == "true", Valid: true},
			OptClean:                  sql.NullBool{Bool: formData.OptClean == "true", Valid: true},
			OptIfExists:               sql.NullBool{Bool: formData.OptIfExists == "true", Valid: true},
			OptCreate:                 sql.NullBool{Bool: formData.OptCreate == "true", Valid: true},
			OptNoComments:             sql.NullBool{Bool: formData.OptNoComments == "true", Valid: true},
			OptSerializableDeferrable: sql.NullBool{Bool: formData.OptSerializableDeferrable == "true", Valid: true},
			ParallelDumpEnabled:       sql.NullBool{Bool: formData.ParallelDumpEnabled == "true", Valid: true},
			Tag:                       sql.NullString{String: tag, Valid: true},
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.AlertWithRefresh(c, "Backup updated")
}

// editBackupButton renders only the trigger and an empty modal: the form
// itself (heavy — the timezone select alone is ~50KB) is fetched lazily when
// the modal opens. Embedding it in every list row bloated the page to ~1MB.
func editBackupButton(backup dbgen.BackupsServicePaginateBackupsRow) nodx.Node {
	mo := component.Modal(component.ModalParams{
		Size:  component.SizeLg,
		Title: "Edit backup",
		Content: []nodx.Node{
			nodx.Div(
				htmx.HxGet(pathutil.BuildPath(
					fmt.Sprintf("/dashboard/backups/%s/edit-form", backup.ID),
				)),
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
			lucide.Pencil(),
			component.SpanText("Edit backup"),
		),
	})
}

func (h *handlers) editBackupFormHandler(c echo.Context) error {
	if !reqctx.GetCtx(c).Access.CanManageApp() {
		return webaccess.ForbiddenHTMX(c)
	}

	backupID, err := uuid.Parse(c.Param("backupID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	backup, err := h.servs.BackupsService.GetBackup(c.Request().Context(), backupID)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, editBackupForm(backup))
}

func editBackupForm(backup dbgen.Backup) nodx.Node {
	yesNoOptions := func(value bool) nodx.Node {
		return nodx.Group(
			nodx.Option(
				nodx.Value("true"),
				nodx.Text(i18n.LabelYes),
				nodx.If(value, nodx.Selected("")),
			),
			nodx.Option(
				nodx.Value("false"),
				nodx.Text(i18n.LabelNo),
				nodx.If(!value, nodx.Selected("")),
			),
		)
	}

	return nodx.FormEl(
		htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/backups/%s/edit", backup.ID))),
		htmx.HxDisabledELT("find button"),
		nodx.Class("space-y-2 text-base"),

		alpine.XData(fmt.Sprintf(`{
			parallel_dump: "%t",
		}`, backup.ParallelDumpEnabled)),

		component.InputControl(component.InputControlParams{
			Name:        "name",
			Label:       i18n.LabelName,
			Placeholder: "My backup",
			Required:    true,
			Type:        component.InputTypeText,
			Children: []nodx.Node{
				nodx.Value(backup.Name),
			},
		}),

		component.InputControl(component.InputControlParams{
			Name:        "cron_expression",
			Label:       "Cron expression",
			Placeholder: "* * * * *",
			Required:    true,
			Type:        component.InputTypeText,
			HelpText:    "Cron expression for the backup schedule",
			Pattern:     `^\S+\s+\S+\s+\S+\s+\S+\s+\S+$`,
			Children: []nodx.Node{
				nodx.Value(backup.CronExpression),
			},
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
						return nodx.Option(
							nodx.Value(tz.TzCode),
							nodx.Text(tz.Label),
							nodx.If(
								tz.TzCode == backup.TimeZone,
								nodx.Selected(""),
							),
						)
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
			HelpButtonChildren: destinationDirectoryHelp(),
			Pattern:            `^\/\S*[^\/]$`,
			Children: []nodx.Node{
				nodx.Value(backup.DestDir),
			},
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
				nodx.Value(fmt.Sprintf("%d", backup.RetentionDays)),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:               "monthly_retention_enabled",
			Label:              "Keep monthly backups",
			Required:           true,
			HelpButtonChildren: monthlyRetentionHelp(),
			Children: []nodx.Node{
				yesNoOptions(backup.MonthlyRetentionEnabled),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:               "parallel_dump_enabled",
			Label:              "Parallel dump",
			Required:           true,
			HelpButtonChildren: parallelDumpHelp(),
			Children: []nodx.Node{
				alpine.XModel("parallel_dump"),
				yesNoOptions(backup.ParallelDumpEnabled),
			},
		}),

		component.InputControl(component.InputControlParams{
			Name:        "tag",
			Label:       "Worker tag",
			Placeholder: "default",
			Type:        component.InputTypeText,
			HelpText:    "Only workers configured with a matching PBW_WORKER_TAGS entry claim this backup. Empty = \"default\"",
			Children: []nodx.Node{
				nodx.Value(backup.Tag),
			},
		}),

		component.SelectControl(component.SelectControlParams{
			Name:     "is_active",
			Label:    "Activate backup",
			Required: true,
			Children: []nodx.Node{
				yesNoOptions(backup.IsActive),
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
						yesNoOptions(backup.OptDataOnly),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:  "opt_schema_only",
					Label: "--schema-only",
					Children: []nodx.Node{
						parallelDumpDisables(),
						yesNoOptions(backup.OptSchemaOnly),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_clean",
					Label:    "--clean",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(backup.OptClean),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_if_exists",
					Label:    "--if-exists",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(backup.OptIfExists),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_create",
					Label:    "--create",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(backup.OptCreate),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:     "opt_no_comments",
					Label:    "--no-comments",
					Required: true,
					Children: []nodx.Node{
						yesNoOptions(backup.OptNoComments),
					},
				}),

				component.SelectControl(component.SelectControlParams{
					Name:  "opt_serializable_deferrable",
					Label: "--serializable-deferrable",
					Children: []nodx.Node{
						parallelDumpDisables(),
						yesNoOptions(backup.OptSerializableDeferrable),
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
				component.SpanText(i18n.BtnSave),
				lucide.Save(),
			),
		),
	)
}
