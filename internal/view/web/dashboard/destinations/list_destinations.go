package destinations

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/destinations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/paginateutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/pathutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func (h *handlers) listDestinationsHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var formData struct {
		Page int `query:"page" validate:"required,min=1"`
	}
	if err := c.Bind(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}
	if err := validate.Struct(&formData); err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	pagination, destinations, err := h.servs.DestinationsService.PaginateDestinations(
		ctx, destinations.PaginateDestinationsParams{
			Page:  formData.Page,
			Limit: 20,
		},
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK, listDestinations(pagination, destinations),
	)
}

func listDestinations(
	pagination paginateutil.PaginateResponse,
	destinations []dbgen.DestinationsServicePaginateDestinationsRow,
) nodx.Node {
	if len(destinations) < 1 {
		return component.EmptyResults(component.EmptyResultsParams{
			Title:    "No destinations found",
			Subtitle: "Destinations will appear here after they are added",
		})
	}

	cards := []nodx.Node{}
	for _, destination := range destinations {
		cards = append(cards, component.ItemCard(
			nil,
			[]nodx.Node{
				component.OptionsDropdown(
					component.OptionsDropdownA(
						nodx.Href(pathutil.BuildPath(
							fmt.Sprintf("/dashboard/jobs?destination=%s", destination.ID),
						)),
						nodx.Target("_blank"),
						lucide.List(),
						component.SpanText(i18n.BtnShowTasks),
					),
					editDestinationButton(destination),
					component.OptionsDropdownButton(
						htmx.HxPost(pathutil.BuildPath(fmt.Sprintf("/dashboard/destinations/%s/test", destination.ID))),
						htmx.HxDisabledELT("this"),
						lucide.PlugZap(),
						component.SpanText(i18n.BtnTestConnection),
					),
					deleteDestinationButton(destination.ID),
				),
				component.HealthStatusPing(
					destination.TestOk, destination.TestError, destination.LastTestAt,
				),
				nodx.SpanEl(nodx.Class("font-semibold flex-1 truncate"), component.SpanText(destination.Name)),
			},
			[]nodx.Node{
				component.Stat("Bucket", nodx.SpanEl(
					nodx.Class("inline-flex items-center space-x-1"),
					component.CopyButtonSm(destination.BucketName),
					component.SpanText(destination.BucketName),
				)),
				component.Stat("Endpoint", nodx.SpanEl(
					nodx.Class("inline-flex items-center space-x-1"),
					component.CopyButtonSm(destination.Endpoint),
					component.SpanText(destination.Endpoint),
				)),
				component.Stat("Region", nodx.SpanEl(
					nodx.Class("inline-flex items-center space-x-1"),
					component.CopyButtonSm(destination.Region),
					component.SpanText(destination.Region),
				)),
				component.Stat("Access key", nodx.SpanEl(
					nodx.Class("inline-flex items-center space-x-1"),
					component.CopyButtonSm(destination.DecryptedAccessKey),
					component.SpanText("**********"),
				)),
				component.Stat("Secret key", nodx.SpanEl(
					nodx.Class("inline-flex items-center space-x-1"),
					component.CopyButtonSm(destination.DecryptedSecretKey),
					component.SpanText("**********"),
				)),
				component.Stat("Added", component.SpanText(
					destination.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
			},
		))
	}

	if pagination.HasNextPage {
		cards = append(cards, nodx.Div(
			htmx.HxGet(pathutil.BuildPath(fmt.Sprintf(
				"/dashboard/destinations/list?page=%d", pagination.NextPage,
			))),
			htmx.HxTrigger("intersect once"),
			htmx.HxSwap("afterend"),
		))
	}

	return component.RenderableGroup(cards)
}
