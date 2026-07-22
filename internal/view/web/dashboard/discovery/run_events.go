package discovery

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	discoveryservice "github.com/kamisamamayuri-cyber/pgwarden/internal/service/discovery"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/respondhtmx"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type listQueryData struct {
	Level    string `query:"level" validate:"omitempty,oneof=info error"`
	Event    string `query:"event" validate:"omitempty,max=64"`
	Host     string `query:"host" validate:"omitempty,max=253"`
	Port     int    `query:"port" validate:"omitempty,min=1,max=65535"`
	Database string `query:"database" validate:"omitempty,max=128"`
	Page     int    `query:"page" validate:"required,min=1"`
}

func (h *handlers) runDetailsHandler(c echo.Context) error {
	return h.runEventsHandler(c, false)
}

func (h *handlers) runReportHandler(c echo.Context) error {
	return h.runEventsHandler(c, true)
}

func (h *handlers) runEventsHandler(c echo.Context, reportOnly bool) error {
	runID, err := uuid.Parse(c.Param("runID"))
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	events, err := h.servs.DiscoveryService.ListRunEvents(
		c.Request().Context(),
		runID,
		reportOnly,
	)
	if err != nil {
		return respondhtmx.ToastError(c, err.Error())
	}

	poll := !reportOnly

	if c.Request().Header.Get("HX-Target") == tbodyID(runID) {
		return echoutil.RenderNodx(c, http.StatusOK, listEventRows(events))
	}

	return echoutil.RenderNodx(c, http.StatusOK, renderRunEvents(runID, events, poll))
}

func tbodyID(runID uuid.UUID) string {
	return "discovery-run-events-tbody-" + runID.String()
}

func renderRunEvents(runID uuid.UUID, events []dbgen.DiscoveryEvent, poll bool) nodx.Node {
	contentID := "discovery-run-events-" + runID.String()
	return nodx.Div(
		nodx.Id(contentID),
		nodx.Class("overflow-auto max-h-[65dvh]"),
		nodx.Table(
			nodx.Class("table text-nowrap"),
			nodx.Thead(
				nodx.Class("sticky top-0 z-[1] bg-base-200"),
				nodx.Tr(
					nodx.Th(component.SpanText("Time")),
					nodx.Th(component.SpanText("Level")),
					nodx.Th(component.SpanText("Event")),
					nodx.Th(component.SpanText("Host")),
					nodx.Th(component.SpanText("Port")),
					nodx.Th(component.SpanText("Database")),
					nodx.Th(component.SpanText("Message")),
				),
			),
			renderRunEventsTbody(runID, events, poll),
		),
	)
}

func renderRunEventsTbody(runID uuid.UUID, events []dbgen.DiscoveryEvent, poll bool) nodx.Node {
	nodes := []nodx.Node{nodx.Id(tbodyID(runID))}
	if poll && runIsRunning(events) {
		nodes = append(nodes,
			htmx.HxGet(buildRunDetailsURL(runID.String())),
			htmx.HxTrigger("every 3s"),
			htmx.HxTarget("this"),
			htmx.HxSwap("innerHTML"),
		)
	}
	nodes = append(nodes, listEventRows(events))
	return nodx.Tbody(nodes...)
}

func listEventRows(events []dbgen.DiscoveryEvent) nodx.Node {
	if len(events) < 1 {
		return component.EmptyResultsTr(component.EmptyResultsParams{
			Title:    "No discovery events found",
			Subtitle: "No events yet for this run",
		})
	}

	trs := make([]nodx.Node, 0, len(events))
	for _, event := range events {
		trs = append(trs, eventRow(event))
	}
	return component.RenderableGroup(trs)
}

func eventRow(event dbgen.DiscoveryEvent) nodx.Node {
	return nodx.Tr(
		nodx.Td(component.SpanText(
			event.CreatedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
		)),
		nodx.Td(discoveryLevelBadge(event.Level)),
		nodx.Td(component.SpanText(event.Event)),
		nodx.Td(component.SpanText(event.Host)),
		nodx.Td(component.SpanText(nullableInt(event.Port))),
		nodx.Td(component.SpanText(nullableString(event.DatabaseName))),
		nodx.Td(
			nodx.Class("whitespace-normal min-w-96"),
			component.SpanText(event.Message),
		),
	)
}

func runIsRunning(events []dbgen.DiscoveryEvent) bool {
	var lastEventTime time.Time
	for _, event := range events {
		if event.Event == "scan_finished" {
			return false
		}
		if event.CreatedAt.After(lastEventTime) {
			lastEventTime = event.CreatedAt
		}
	}
	if lastEventTime.IsZero() {
		return true
	}
	return time.Since(lastEventTime) < 30*time.Minute
}

func paginateParamsFromQuery(q listQueryData) discoveryservice.PaginateEventsParams {
	return discoveryservice.PaginateEventsParams{
		Page:  q.Page,
		Limit: 30,
		LevelFilter: sql.NullString{
			String: q.Level, Valid: q.Level != "",
		},
		EventFilter: sql.NullString{
			String: q.Event, Valid: q.Event != "",
		},
		HostFilter: sql.NullString{
			String: q.Host, Valid: q.Host != "",
		},
		PortFilter: sql.NullInt32{
			Int32: int32(q.Port), Valid: q.Port > 0,
		},
		DatabaseFilter: sql.NullString{
			String: q.Database, Valid: q.Database != "",
		},
	}
}

func discoveryLevelBadge(level string) nodx.Node {
	class := "badge badge-sm "
	switch level {
	case "error":
		class += "badge-error"
	case "warn":
		class += "badge-warning"
	default:
		class += "badge-info"
	}
	return nodx.SpanEl(nodx.Class(class), nodx.Text(level))
}

func nullableString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableInt(value sql.NullInt32) string {
	if !value.Valid {
		return ""
	}
	return strconv.Itoa(int(value.Int32))
}
