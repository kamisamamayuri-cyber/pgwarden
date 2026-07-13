package restorations

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

type resQueryData struct {
	Execution uuid.UUID `query:"execution" validate:"omitempty,uuid"`
	Database  uuid.UUID `query:"database" validate:"omitempty,uuid"`
	Status    string    `query:"status" validate:"omitempty,oneof=queued running success failed"`
}

func (h *handlers) indexPageHandler(c echo.Context) error {
	reqCtx := reqctx.GetCtx(c)

	var queryData resQueryData
	if err := c.Bind(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	if err := validate.Struct(&queryData); err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return echoutil.RenderNodx(c, http.StatusOK, indexPage(reqCtx, queryData))
}

func indexPage(reqCtx reqctx.Ctx, queryData resQueryData) nodx.Node {
	filterQuery := resFilterQuery(queryData)

	content := []nodx.Node{
		nodx.Div(
			component.H1Text("Restorations"),
			component.PText(`
				History of database restore operations from backups.
			`),
		),
		component.CardBox(component.CardBoxParams{
			Class: "mt-4",
			Children: []nodx.Node{
				restorationStatusFilters(filterQuery),
				nodx.Div(
					nodx.Class("overflow-x-auto"),
					nodx.Table(
						nodx.Class("table text-nowrap"),
						nodx.Thead(
							nodx.Tr(
								nodx.Th(nodx.Class("w-1")),
								nodx.Th(component.SpanText(i18n.LabelStatus)),
								nodx.Th(component.SpanText(i18n.LabelBackup)),
								nodx.Th(component.SpanText("Target DB")),
								nodx.Th(component.SpanText(i18n.LabelTask)),
								nodx.Th(component.SpanText(i18n.LabelStartedAt)),
								nodx.Th(component.SpanText(i18n.LabelFinishedAt)),
								nodx.Th(component.SpanText(i18n.LabelDuration)),
							),
						),
						nodx.Tbody(
							component.SkeletonTr(8),
							htmx.HxGet(buildRestorationsListURL(filterQuery, 1)),
							htmx.HxTrigger("load"),
						),
					),
				),
			},
		}),
	}

	return layout.Dashboard(reqCtx, layout.DashboardParams{
		Title: "Restorations",
		Body:  content,
	})
}

func restorationStatusFilters(q resFilterQuery) nodx.Node {
	filters := []struct {
		label  string
		status string
	}{
		{label: "All", status: ""},
		{label: i18n.StatusLabel("queued"), status: "queued"},
		{label: i18n.StatusLabel("running"), status: "running"},
		{label: i18n.StatusLabel("success"), status: "success"},
		{label: i18n.StatusLabel("failed"), status: "failed"},
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
			nodx.Href(buildRestorationsIndexURL(q, filter.status)),
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
