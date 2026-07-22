package backups

import (
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/component"
	nodx "github.com/nodxdev/nodxgo"
	lucide "github.com/nodxdev/nodxgo-lucide"
)

func localBackupsHelp() []nodx.Node {
	return []nodx.Node{
		component.H3Text("Local backups"),
		component.PText(`
			Local backups are stored on the server where PG Warden runs.
			They are saved to the /backups directory — you can mount a Docker volume
			for persistent storage.
		`),

		nodx.Div(
			nodx.Class("mt-2"),
			component.H3Text("Remote backups"),
			component.PText(`
				Remote backups are stored in an S3-compatible storage destination.
				No Docker volumes need to be configured for local storage.
			`),
		),
	}
}

func cronExpressionHelp() []nodx.Node {
	return []nodx.Node{
		component.PText(`
			A cron expression defines the schedule for periodic tasks in Unix systems.
			Five fields: minute, hour, day of month, month, day of week.
		`),

		nodx.Div(
			nodx.Class("mt-4 flex justify-end items-center space-x-1"),
			nodx.A(
				nodx.Href("https://en.wikipedia.org/wiki/Cron"),
				nodx.Target("_blank"),
				nodx.Class("btn btn-ghost"),
				component.SpanText("Learn more"),
				lucide.ExternalLink(),
			),
			nodx.A(
				nodx.Href("https://crontab.guru/examples.html"),
				nodx.Target("_blank"),
				nodx.Class("btn btn-ghost"),
				component.SpanText("Examples and common expressions"),
				lucide.ExternalLink(),
			),
		),
	}
}

func timezoneFilenamesHelp() []nodx.Node {
	serverTimezone := time.Now().Location().String()

	return []nodx.Node{
		component.PText(`
			The timezone in which the cron expression is evaluated.
		`),
		nodx.P(
			component.SpanText(`
				Backup file names use the server timezone (currently
			`),
			component.BText(serverTimezone),
			component.SpanText(")."),
		),

		nodx.Div(
			nodx.Class("mt-4 flex justify-end items-center"),
			nodx.A(
				nodx.Href(component.RepoURL+"/-/blob/master/README.md"),
				nodx.Target("_blank"),
				nodx.Class("btn btn-ghost"),
				component.SpanText("Learn more in project README"),
				lucide.ExternalLink(),
			),
		),
	}
}

func destinationDirectoryHelp() []nodx.Node {
	return []nodx.Node{
		component.PText(`
			Directory where backups are stored, relative to the destination base directory.
			Must start with "/" and must not contain spaces or end with "/".
		`),

		nodx.Div(
			nodx.Class("mt-2"),
			component.H3Text("Local backups"),
			component.PText(`
				For local backups the base directory is /backups. Files are stored at:
			`),
			nodx.Div(
				nodx.ClassMap{
					"whitespace-nowrap p-1": true,
					"overflow-x-scroll":     true,
					"font-mono":             true,
				},
				component.BText(
					"/backups/<destination-directory>/<YYYY>/<MM>/<DD>/dump-<random-suffix>.zip",
				),
			),
		),

		nodx.Div(
			nodx.Class("mt-2"),
			component.H3Text("Remote backups"),
			component.PText(`
				For remote backups the base directory is the bucket root. Files are stored at:
			`),
			nodx.Div(
				nodx.ClassMap{
					"whitespace-nowrap p-1": true,
					"overflow-x-scroll":     true,
					"font-mono":             true,
				},
				component.BText(
					"s3://<bucket>/<destination-directory>/<YYYY>/<MM>/<DD>/dump-<random-suffix>.zip",
				),
			),
		),
	}
}

func retentionDaysHelp() []nodx.Node {
	return []nodx.Node{
		nodx.Div(
			nodx.Class("space-y-2"),

			component.PText(`
				Retention period — number of days before backup files are automatically deleted.
				Old backups are removed to save space. The period is evaluated per execution.
			`),

			component.PText(`
				Setting 0 applies the global execution retention period
				(default 30 days, see PBW_EXECUTION_RETENTION_DAYS).
			`),
		),
	}
}

func monthlyRetentionHelp() []nodx.Node {
	return []nodx.Node{
		nodx.Div(
			nodx.Class("space-y-2"),

			component.PText(`
				When enabled, the first successful execution of each calendar month is
				kept for PBW_MONTHLY_RETENTION_MONTHS months (server-wide setting,
				default 12), independent of the retention period above. All other,
				more frequent executions of this backup still expire after the regular
				retention period.
			`),

			component.PText(`
				Example: retention of 30 days with monthly backups enabled keeps one
				snapshot per month for 12 months, while daily copies are pruned after
				30 days.
			`),
		),
	}
}

func pgDumpOptionsHelp() []nodx.Node {
	return []nodx.Node{
		nodx.Div(
			nodx.Class("space-y-2"),

			component.PText(`
				Backups are created with pg_dump, which produces consistent snapshots
				even while the database is in active use.
			`),

			component.PText(`
				Options passed to pg_dump. By default PG Warden passes no options,
				producing a full backup.
			`),

			nodx.Div(
				nodx.Class("flex justify-end"),
				nodx.A(
					nodx.Class("btn btn-ghost"),
					nodx.Href("https://www.postgresql.org/docs/current/app-pgdump.html"),
					nodx.Target("_blank"),
					component.SpanText("pg_dump documentation"),
					lucide.ExternalLink(nodx.Class("ml-1")),
				),
			),
		),
	}
}

func serializableDeferrableHelp() []nodx.Node {
	return []nodx.Node{
		component.PText(`
			Runs pg_dump in a SERIALIZABLE DEFERRABLE transaction. PostgreSQL defers the
			transaction start until a clean snapshot can be taken without conflicts —
			the dump does not acquire AccessShareLock on tables and does not block
			concurrent write operations.
		`),
		component.PText(`
			Recommended for hosts without streaming replication replicas, where a regular
			pg_dump may compete for locks with DDL operations (ALTER TABLE, VACUUM FULL).
			On hosts with a replica it is preferable to dump from the replica instead.
		`),
		component.PText(`
			Downside: under heavy write load the transaction may wait a long time
			for a suitable window before starting.
		`),
	}
}

func parallelDumpDisables() nodx.Node {
	return nodx.Group(
		nodx.Attr(":disabled", `parallel_dump === "true"`),
		nodx.Attr("x-effect", `if (parallel_dump === "true") $el.value = "false"`),
	)
}

func parallelDumpHelp() []nodx.Node {
	return []nodx.Node{
		component.PText(`
			Dumps tables with several pg_dump processes sharing one consistent
			snapshot, instead of a single sequential pass. Speeds up databases
			with many tables and works around per-connection network throughput
			limits. The archive becomes multi-file (one SQL file per table plus
			schema sections) with join/restore scripts included; restores work
			with both formats.
		`),
		component.PText(`
			The number of concurrent pg_dump processes is set server-wide by
			PBW_DUMP_PARALLEL_JOBS. Each job opens its own connection to the
			source database.
		`),
		component.PText(`
			Not compatible with --data-only, --schema-only, or
			--serializable-deferrable: such backups fall back to the classic
			single-file dump.
		`),
	}
}
