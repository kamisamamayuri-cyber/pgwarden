package executions

import (
	"net/http"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/echoutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/validate"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/reqctx"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/layout"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type execsQueryData struct {
	Database    uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Destination uuid.UUID `query:"destination" validate:"omitempty,uuid"`
	Backup      uuid.UUID `query:"backup" validate:"omitempty,uuid"`
	Status      string    `query:"status" validate:"omitempty,oneof=queued running success failed deleted"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData execsQueryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func indexPage(reqCtx reqctx.Ctx, queryData execsQueryData) nodx.Node {
	filterQuery := execsFilterQuery(queryData)

	content := []nodx.Node{
		component.H1Text("Executions"),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				executionStatusFilters(filterQuery),
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText(i18n.LabelStatus)),
								nodx.Th(component.SpanText(i18n.LabelBackup)),
								nodx.Th(component.SpanText(i18n.LabelDatabase)),
								nodx.Th(component.SpanText(i18n.LabelDestination)),
								nodx.Th(component.SpanText(i18n.LabelStartedAt)),
								nodx.Th(component.SpanText(i18n.LabelFinishedAt)),
								nodx.Th(component.SpanText(i18n.LabelDuration)),
								nodx.Th(component.SpanText(i18n.LabelFileSize)),
							),
						),
						nodx.Tbody(
							component.SkeletonTr(8),
							htmx.HxGet(buildExecutionsListURL(filterQuery, 1)),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Executions",
		Body:  content,
	})
}

func executionStatusFilters(q execsFilterQuery) nodx.Node {
	filters := []struct {
		label  string
		status string
	}{
		{label: "All", status: ""},
		{label: i18n.StatusLabel("queued"), status: "queued"},
		{label: i18n.StatusLabel("running"), status: "running"},
		{label: i18n.StatusLabel("success"), status: "success"},
		{label: i18n.StatusLabel("failed"), status: "failed"},
		{label: i18n.StatusLabel("deleted"), status: "deleted"},
	}

	buttons := make([]nodx.Node, 0, len(filters))
	for _, filter := range filters {
		active := q.Status == filter.status
		btnClass := "btn btn-sm"
		if active {
			btnClass += " btn-primary"
		} else {
			btnClass += " btn-ghost"
		}

		buttons = append(buttons, nodx.A(
			nodx.Class(btnClass),
			nodx.Href(buildExecutionsIndexURL(q, filter.status)),
			nodx.Text(filter.label),
		))
	}

	return nodx.Div(
		nodx.Class("mb-4 flex flex-wrap gap-2"),
		nodx.SpanEl(
			nodx.Class("text-sm text-base-content/70 self-center mr-1"),
			nodx.Text(i18n.LabelStatus+":"),
		),
		component.RenderableGroup(buttons),
	)
}
