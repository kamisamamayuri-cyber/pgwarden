package restorations

import (
	"database/sql"
	"strings"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/database/dbgen"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/service/restorations"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/util/timeutil"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/google/uuid"
	nodx "github.com/nodxdev/nodxgo"
	htmx "github.com/nodxdev/nodxgo-htmx"
)

type restorationDetailsView struct {
	ID                 uuid.UUID
	Status             string
	BackupName         string
	TargetDatabaseName sql.NullString
	DatabaseName       sql.NullString
	Message            sql.NullString
	LogTail            sql.NullString
	StartedAt          sql.NullTime
	FinishedAt         sql.NullTime
}

func restorationDetailsViewFromPaginate(
	row dbgen.RestorationsServicePaginateRestorationsRow,
) restorationDetailsView {
	return restorationDetailsView{
		ID:                 row.ID,
		Status:             row.Status,
		BackupName:         row.BackupName,
		TargetDatabaseName: row.TargetDatabaseName,
		DatabaseName:       row.DatabaseName,
		Message:            row.Message,
		LogTail:            row.LogTail,
		StartedAt:          sql.NullTime{Time: row.StartedAt, Valid: true},
		FinishedAt:         row.FinishedAt,
	}
}

func restorationDetailsViewFromGet(
	row dbgen.RestorationsServiceGetRestorationRow,
) restorationDetailsView {
	return restorationDetailsView{
		ID:                 row.ID,
		Status:             row.Status,
		BackupName:         row.BackupName,
		TargetDatabaseName: row.TargetDatabaseName,
		DatabaseName:       row.DatabaseName,
		Message:            row.Message,
		LogTail:            row.LogTail,
		StartedAt:          sql.NullTime{Time: row.StartedAt, Valid: true},
		FinishedAt:         row.FinishedAt,
	}
}

func restorationModalID(id uuid.UUID) string {
	return "restoration-modal-" + id.String()
}

func restorationDetailsLoadingID(id uuid.UUID) string {
	return "restoration-details-loading-" + id.String()
}

func renderRestorationDetails(v restorationDetailsView, poll bool) nodx.Node {
	detailsID := "restoration-details-" + v.ID.String()

	nodes := []nodx.Node{
		nodx.Id(detailsID),
		nodx.Class("overflow-x-auto"),
		restorationDetailsTable(v),
	}
	if poll {
		openEvent := restorationModalID(v.ID) + "_open from:window"
		nodes = append([]nodx.Node{
			htmx.HxGet(buildRestorationDetailsURL(v.ID)),
			htmx.HxTrigger(openEvent + ", every 3s"),
			htmx.HxSwap("outerHTML"),
			htmx.HxIndicator("#" + restorationDetailsLoadingID(v.ID)),
		}, nodes...)
	}

	return nodx.Div(nodes...)
}

func restorationDetailsTable(v restorationDetailsView) nodx.Node {
	logLines := []string{}
	if v.LogTail.Valid {
		logLines = restorations.LogTailToLines(v.LogTail.String)
	}

	startedAt := v.StartedAt.Time
	if !v.StartedAt.Valid {
		startedAt = v.FinishedAt.Time
	}

	return nodx.Table(
		nodx.Class("table [&_th]:text-nowrap"),
		nodx.Tr(
			nodx.Th(component.SpanText("ID")),
			nodx.Td(component.SpanText(v.ID.String())),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelStatus)),
			nodx.Td(component.StatusBadge(v.Status)),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelBackup)),
			nodx.Td(component.SpanText(v.BackupName)),
		),
		nodx.Tr(
			nodx.Th(component.SpanText("Target DB")),
			nodx.Td(
				nodx.Class("break-all"),
				component.SpanText(restorationTargetDatabaseFromView(v)),
			),
		),
		nodx.If(
			v.Message.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelMessage)),
				nodx.Td(
					nodx.Class("break-all"),
					component.SpanText(v.Message.String),
				),
			),
		),
		nodx.If(
			len(logLines) > 0,
			nodx.Tr(
				nodx.Th(component.SpanText("Log (last lines)")),
				nodx.Td(
					nodx.Pre(
						nodx.Class(
							"text-xs whitespace-pre-wrap break-all max-h-40 overflow-y-auto",
						),
						nodx.Text(strings.Join(logLines, "\n")),
					),
				),
			),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelStartedAt)),
			nodx.Td(component.SpanText(
				startedAt.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
			)),
		),
		nodx.Tr(
			nodx.Th(component.SpanText(i18n.LabelDuration)),
			nodx.Td(component.SpanText(
				restorations.RestorationDuration(
					startedAt, v.FinishedAt,
				),
			)),
		),
		nodx.If(
			v.FinishedAt.Valid,
			nodx.Tr(
				nodx.Th(component.SpanText(i18n.LabelFinishedAt)),
				nodx.Td(component.SpanText(
					v.FinishedAt.Time.Local().Format(timeutil.LayoutYYYYMMDDHHMMSSPretty),
				)),
			),
		),
	)
}

func restorationTargetDatabaseFromView(v restorationDetailsView) string {
	if v.TargetDatabaseName.Valid && v.TargetDatabaseName.String != "" {
		return v.TargetDatabaseName.String
	}
	if v.DatabaseName.Valid {
		return v.DatabaseName.String
	}
	return "Other database"
}

func restorationTargetDatabase(
	restoration dbgen.RestorationsServicePaginateRestorationsRow,
) string {
	return restorationTargetDatabaseFromView(restorationDetailsViewFromPaginate(restoration))
}
