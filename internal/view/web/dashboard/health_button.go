package dashboard

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
)

func healthButtonHandler(servs *service.Service) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := c.Request().Context()

		databasesQty, err := servs.DatabasesService.GetDatabasesQty(ctx)
		if err != nil {
			return respondhtmx.ToastError(c, err.Error())
		}
		destinationsQty, err := servs.DestinationsService.GetDestinationsQty(ctx)
		if err != nil {
			return respondhtmx.ToastError(c, err.Error())
		}

		return echoutil.RenderNodx(c, http.StatusOK, healthButton(
			databasesQty, destinationsQty,
		))
	}
}

func healthButton(
	databasesQty dbgen.DatabasesServiceGetDatabasesQtyRow,
	destinationsQty dbgen.DestinationsServiceGetDestinationsQtyRow,
) nodx.Node {
	areDatabasesHealthy := databasesQty.Unhealthy == 0
	areDestinationsHealthy := destinationsQty.Unhealthy == 0
	isHealthy := areDatabasesHealthy && areDestinationsHealthy

	pingColor := component.ColorSuccess
	if !isHealthy {
		pingColor = component.ColorError
	}

	mo := component.Modal(component.ModalParams{
		Size:  component.SizeMd,
		Title: "System health",
		Content: []nodx.Node{
			component.PText(`
				Databases and destinations are checked automatically every 10 minutes,
				on PG Warden startup, and when clicking "Test connection" on each
				resource. Details and error messages are available by clicking the
				indicator on the resource.
			`),
			nodx.Table(
				nodx.Class("table mt-2"),
				nodx.Thead(
					nodx.Tr(
						nodx.Th(component.SpanText("Resource")),
						nodx.Th(component.SpanText("Total")),
						nodx.Th(component.SpanText("Available")),
						nodx.Th(component.SpanText("Unavailable")),
					),
				),
				nodx.Tbody(
					nodx.Tr(
						nodx.Td(component.SpanText("Databases")),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", databasesQty.All))),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", databasesQty.Healthy))),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", databasesQty.Unhealthy))),
					),
					nodx.Tr(
						nodx.Td(component.SpanText("Destinations")),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", destinationsQty.All))),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", destinationsQty.Healthy))),
						nodx.Td(component.SpanText(fmt.Sprintf("%d", destinationsQty.Unhealthy))),
					),
				),
			),
		},
	})

	return nodx.Div(
		nodx.Class("inline-block"),
		mo.HTML,
		nodx.Button(
			mo.OpenerAttr,
			nodx.Class("btn btn-ghost btn-neutral"),
			component.SpanText("System health"),
			component.Ping(pingColor),
		),
	)
}
