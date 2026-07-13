package summary

import (
	"fmt"
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
)

func (h *handlers) indexPageHandler(c echo.Context) error {
	ctx := c.Request().Context()
	reqCtx := reqctx.GetCtx(c)

	databasesQty, err := h.servs.DatabasesService.GetDatabasesQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	destinationsQty, err := h.servs.DestinationsService.GetDestinationsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	backupsQty, err := h.servs.BackupsService.GetBackupsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	executionsQty, err := h.servs.ExecutionsService.GetExecutionsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	restorationsQty, err := h.servs.RestorationsService.GetRestorationsQty(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return echoutil.RenderNodx(
		c, http.StatusOK,
		indexPage(
			reqCtx, databasesQty, destinationsQty, backupsQty, executionsQty,
			restorationsQty,
		),
	)
}

func indexPage(
	reqCtx reqctx.Ctx,
	databasesQty dbgen.DatabasesServiceGetDatabasesQtyRow,
	destinationsQty dbgen.DestinationsServiceGetDestinationsQtyRow,
	backupsQty dbgen.BackupsServiceGetBackupsQtyRow,
	executionsQty dbgen.ExecutionsServiceGetExecutionsQtyRow,
	restorationsQty dbgen.RestorationsServiceGetRestorationsQtyRow,
) nodx.Node {
	type ChartData struct {
		Label    string
		Labels   []string
		Data     []int32
		BgColors []string
	}

	countCard := func(
		title string,
		count int64,
		chartData ChartData,
	) nodx.Node {
		chart := func() nodx.Node {
			notAvailable := nodx.Div(
				nodx.Class("size-[218px] flex flex-col justify-center items-center"),
				nodx.SpanEl(
					nodx.Class("text-sm text-base-content pb-[32px]"),
					nodx.Text("No chart data available"),
				),
			)

			if len(chartData.Data) < 1 {
				return notAvailable
			}

			areAllZero := true
			for _, d := range chartData.Data {
				if d != 0 {
					areAllZero = false
					break
				}
			}

			if areAllZero {
				return notAvailable
			}

			chartID := "chart-" + uuid.NewString()
			labels := ""
			for _, label := range chartData.Labels {
				labels += fmt.Sprintf("'%s',", label)
			}

			data := ""
			for _, d := range chartData.Data {
				data += fmt.Sprintf("%d,", d)
			}

			bgColors := ""
			for _, color := range chartData.BgColors {
				bgColors += fmt.Sprintf("'%s',", color)
			}

			return nodx.Div(
				nodx.Class("mt-2"),
				nodx.Div(nodx.Canvas(nodx.Id(chartID))),
				nodx.Script(nodx.Raw(`
					new Chart(document.getElementById('`+chartID+`'), {
						type: 'doughnut',
						data: {
							labels: [`+labels+`],
							datasets: [{
								label: '`+chartData.Label+`',
								data: [`+data+`],
								backgroundColor: [`+bgColors+`],
								borderColor: 'rgba(0, 0, 0, 0)',
								borderWidth: 0
							}]
						},
						options: {
							plugins: {
								legend: {
									position: 'bottom'
								}
							}
						}
					});
				`)),
			)
		}

		return component.CardBox(component.CardBoxParams{
			Class: "flex-none text-center w-[250px]",
			Children: []nodx.Node{
				component.H2Text(fmt.Sprintf("%d %s", count, title)),
				chart(),
			},
		})
	}

	const (
		greenColor  = "#15803D"
		redColor    = "#FF001F"
		yellowColor = "#CA8A04"
		blueColor   = "#2563EB"
	)

	content := []nodx.Node{
		nodx.Div(
			component.H1Text("Overview"),
		),
		nodx.Div(
			nodx.Class("mt-4 flex justify-start flex-wrap gap-4"),

			countCard("Databases", databasesQty.All, ChartData{
				Label:    "Status",
				Labels:   []string{"Available", "Unavailable"},
				Data:     []int32{databasesQty.Healthy, databasesQty.Unhealthy},
				BgColors: []string{greenColor, redColor},
			}),
			countCard("Destinations", destinationsQty.All, ChartData{
				Label:    "Status",
				Labels:   []string{"Available", "Unavailable"},
				Data:     []int32{destinationsQty.Healthy, destinationsQty.Unhealthy},
				BgColors: []string{greenColor, redColor},
			}),
			countCard("Backups", backupsQty.All, ChartData{
				Label:    "Status",
				Labels:   []string{"Active", "Inactive"},
				Data:     []int32{backupsQty.Active, backupsQty.Inactive},
				BgColors: []string{greenColor, redColor},
			}),
			countCard("Executions", executionsQty.All, ChartData{
				Label:  "Status",
				Labels: []string{"Running", "Success", "Error", "Deleted"},
				Data: []int32{
					executionsQty.Running, executionsQty.Success, executionsQty.Failed,
					executionsQty.Deleted,
				},
				BgColors: []string{blueColor, greenColor, redColor, yellowColor},
			}),
			countCard("Restorations", restorationsQty.All, ChartData{
				Label:  "Status",
				Labels: []string{"Running", "Success", "Error"},
				Data: []int32{
					restorationsQty.Running, restorationsQty.Success,
					restorationsQty.Failed,
				},
				BgColors: []string{blueColor, greenColor, redColor},
			}),
		),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Overview",
		Body:  content,
	})
}
