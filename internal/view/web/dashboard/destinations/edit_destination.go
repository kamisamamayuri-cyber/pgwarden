package destinations

import (
	"database/sql"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) editDestinationHandler(c echo.Context) error {
	ctx := c.Request().Context()

	destinationID, err := uuid.Parse(c.Param("destinationID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	var formData createDestinationDTO
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	_, err = h.servs.DestinationsService.UpdateDestination(
		ctx, dbgen.DestinationsServiceUpdateDestinationParams{
			ID:         destinationID,
			Name:       sql.NullString{String: formData.Name, Valid: true},
			BucketName: sql.NullString{String: formData.BucketName, Valid: true},
			Region:     sql.NullString{String: formData.Region, Valid: true},
			Endpoint:   sql.NullString{String: formData.Endpoint, Valid: true},
			AccessKey:  sql.NullString{String: formData.AccessKey, Valid: true},
			SecretKey:  sql.NullString{String: formData.SecretKey, Valid: true},
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return respondhtmx.AlertWithRefresh(c, "Destination updated")
}

func editDestinationButton(
	destination dbgen.DestinationsServicePaginateDestinationsRow,
) nodx.Node {
	idPref := "edit-destination-" + destination.ID.String()
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
		Title: "Edit destination",
		Content: []nodx.Node{
			nodx.FormEl(
				nodx.Id(formID),
				nodx.Class("space-y-2"),

				component.InputControl(component.InputControlParams{
					Name:        "name",
					Label:       i18n.LabelName,
					Placeholder: "My destination",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "A name for easy identification of the destination",
					Children: []nodx.Node{
						nodx.Value(destination.Name),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "bucket_name",
					Label:       "Bucket name",
					Placeholder: "my-bucket",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.BucketName),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "endpoint",
					Label:       i18n.LabelEndpoint,
					Placeholder: "s3-us-west-1.amazonaws.com",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.Endpoint),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "region",
					Label:       "Region",
					Placeholder: "us-west-1",
					Required:    true,
					Type:        component.InputTypeText,
					Children: []nodx.Node{
						nodx.Value(destination.Region),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "access_key",
					Label:       "Access key",
					Placeholder: "Access key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "Stored with PGP encryption.",
					Children: []nodx.Node{
						nodx.Value(destination.DecryptedAccessKey),
					},
				}),

				component.InputControl(component.InputControlParams{
					Name:        "secret_key",
					Label:       "Secret key",
					Placeholder: "Secret key",
					Required:    true,
					Type:        component.InputTypeText,
					HelpText:    "Stored with PGP encryption.",
					Children: []nodx.Node{
						nodx.Value(destination.DecryptedSecretKey),
					},
				}),
			),

			nodx.Div(
				nodx.Class("flex justify-between items-center pt-4"),
				nodx.Div(
					nodx.Button(
						htmxAttributes("/dashboard/destinations/test"),
						nodx.ClassMap{
							btnClass:                      true,
							"btn btn-neutral btn-outline": true,
						},
						nodx.Type("button"),
						component.SpanText(i18n.BtnTestConnection),
						lucide.PlugZap(),
					),
				),
				nodx.Div(
					nodx.Class("flex justify-end items-center space-x-2"),
					component.HxLoadingMd(loadingID),
					nodx.Button(
						htmxAttributes("/dashboard/destinations/"+destination.ID.String()+"/edit"),
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
			component.SpanText("Edit destination"),
		),
	})
}
