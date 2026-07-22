package postgres

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/lib/pq"
)

const maintenanceDatabase = "postgres"

// ownerReassignSystemNamespaceFilter excludes PostgreSQL system namespaces whose
// objects cannot be reassigned to an application role (pg_catalog, pg_toast_<oid>, …).
const ownerReassignSystemNamespaceFilter = `n.nspname <> 'information_schema' AND n.nspname NOT LIKE 'pg_%%'`

// TargetPrepareParams configures drop/recreate of a target database before restore.
type TargetPrepareParams struct {
	DatabaseName string
	Owner        string
}

func (Client) PrepareTargetDatabase(
	ctx context.Context, version PGVersion, connString string,
	params TargetPrepareParams, log io.Writer,
) error {
	info, err := ParseConnString(connString)
	if err != nil {
		return err
	}

	dbName := strings.TrimSpace(params.DatabaseName)
	if dbName == "" {
		dbName = info.Database
	}
	if dbName == "" {
		return fmt.Errorf("target database name is required")
	}
	if dbName == maintenanceDatabase {
		return fmt.Errorf("refusing to drop maintenance database %q", maintenanceDatabase)
	}

	maintenanceConn := info.WithDatabase(maintenanceDatabase)
	quotedDB := pq.QuoteIdentifier(dbName)

	terminateSQL := fmt.Sprintf(
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = %s AND pid <> pg_backend_pid();`,
		pq.QuoteLiteral(dbName),
	)
	if err := runPSQLCommand(ctx, version, maintenanceConn, terminateSQL, log); err != nil {
		return fmt.Errorf("terminate connections: %w", err)
	}

	dropSQL := fmt.Sprintf(`DROP DATABASE IF EXISTS %s;`, quotedDB)
	if err := runPSQLCommand(ctx, version, maintenanceConn, dropSQL, log); err != nil {
		return fmt.Errorf("drop database: %w", err)
	}

	createSQL := fmt.Sprintf(`CREATE DATABASE %s`, quotedDB)
	owner := strings.TrimSpace(params.Owner)
	if owner != "" {
		createSQL += fmt.Sprintf(" OWNER %s", pq.QuoteIdentifier(owner))
	}
	createSQL += ";"
	if err := runPSQLCommand(ctx, version, maintenanceConn, createSQL, log); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	return nil
}

// PreflightTarget checks, before any destructive step runs, that the target
// host is reachable and (when owner is set) that the owner role already
// exists on the cluster. Both checks run against the maintenance database
// ("postgres") since the target application database may not exist yet —
// PrepareTargetDatabase creates it later. Free disk space cannot be checked
// here: PG Warden has no shell/VM access to the target host, only a
// PostgreSQL connection.
func (Client) PreflightTarget(
	ctx context.Context, version PGVersion, connString, owner string,
) error {
	info, err := ParseConnString(connString)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}
	if _, ok := info.Extra["connect_timeout"]; !ok {
		info.Extra["connect_timeout"] = "30"
	}
	maintenanceConn := info.WithDatabase(maintenanceDatabase)

	owner = strings.TrimSpace(owner)
	if owner == "" {
		cmd := exec.CommandContext(ctx, version.Value.PSQL, maintenanceConn, "-c", "SELECT 1;")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("target host unreachable: %s", strings.TrimSpace(string(output)))
		}
		return nil
	}

	cmd := exec.CommandContext(
		ctx, version.Value.PSQL, maintenanceConn,
		"-tAc", fmt.Sprintf("SELECT 1 FROM pg_roles WHERE rolname = %s", pq.QuoteLiteral(owner)),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("target host unreachable: %s", strings.TrimSpace(string(output)))
	}
	if strings.TrimSpace(string(output)) != "1" {
		return fmt.Errorf("owner role %q does not exist on target cluster", owner)
	}
	return nil
}

func (Client) ReassignDatabaseOwner(
	ctx context.Context, version PGVersion, connString, owner string, log io.Writer,
) error {
	owner = strings.TrimSpace(owner)
	if owner == "" {
		return nil
	}

	info, err := ParseConnString(connString)
	if err != nil {
		return err
	}

	quotedOwner := pq.QuoteIdentifier(owner)
	quotedDB := pq.QuoteIdentifier(info.Database)
	ownerLit := pq.QuoteLiteral(owner)

	sql := fmt.Sprintf(`
ALTER DATABASE %s OWNER TO %s;
ALTER SCHEMA public OWNER TO %s;

SELECT format('ALTER SCHEMA %%I OWNER TO %%I', n.nspname, %s::text)
FROM pg_namespace n
WHERE %s
  AND pg_get_userbyid(n.nspowner) IS NOT NULL
  AND pg_get_userbyid(n.nspowner) <> %s::text
ORDER BY n.nspname
\gexec

SELECT format(
  'ALTER %%s %%I.%%I OWNER TO %%I',
  CASE c.relkind
    WHEN 'v' THEN 'VIEW'
    WHEN 'm' THEN 'MATERIALIZED VIEW'
    WHEN 'S' THEN 'SEQUENCE'
    WHEN 'f' THEN 'FOREIGN TABLE'
    WHEN 'i' THEN 'INDEX'
    WHEN 'c' THEN 'TYPE'
    ELSE 'TABLE'
  END,
  n.nspname, c.relname, %s::text
)
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE %s
  AND pg_get_userbyid(c.relowner) IS NOT NULL
  AND pg_get_userbyid(c.relowner) <> %s::text
  AND c.relkind IN ('r', 'p', 'v', 'm', 'S', 'f', 'i', 'c')
ORDER BY
  CASE c.relkind
    WHEN 'r' THEN 1
    WHEN 'p' THEN 1
    WHEN 'f' THEN 1
    WHEN 'S' THEN 2
    WHEN 'v' THEN 3
    WHEN 'm' THEN 3
    WHEN 'i' THEN 4
    WHEN 'c' THEN 5
    ELSE 99
  END,
  n.nspname,
  c.relname
\gexec

SELECT format('ALTER DOMAIN %%I.%%I OWNER TO %%I', n.nspname, t.typname, %s::text)
FROM pg_type t
JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'd'
  AND %s
  AND pg_get_userbyid(t.typowner) IS NOT NULL
  AND pg_get_userbyid(t.typowner) <> %s::text
ORDER BY n.nspname, t.typname
\gexec

SELECT format(
  'ALTER %%s %%I.%%I(%%s) OWNER TO %%I',
  CASE p.prokind WHEN 'p' THEN 'PROCEDURE' WHEN 'a' THEN 'AGGREGATE' ELSE 'FUNCTION' END,
  n.nspname, p.proname, pg_get_function_identity_arguments(p.oid), %s::text
)
FROM pg_proc p
JOIN pg_namespace n ON n.oid = p.pronamespace
WHERE %s
  AND pg_get_userbyid(p.proowner) IS NOT NULL
  AND pg_get_userbyid(p.proowner) <> %s::text
ORDER BY n.nspname, p.proname
\gexec
`,
		quotedDB, quotedOwner, quotedOwner,
		ownerLit, ownerReassignSystemNamespaceFilter, ownerLit,
		ownerLit, ownerReassignSystemNamespaceFilter, ownerLit,
		ownerLit, ownerReassignSystemNamespaceFilter, ownerLit,
		ownerLit, ownerReassignSystemNamespaceFilter, ownerLit)

	return runPSQLScript(ctx, version, connString, sql, log)
}

func runPSQLScript(
	ctx context.Context, version PGVersion, connString, script string, log io.Writer,
) error {
	errorBuffer := &bytes.Buffer{}
	var stderr io.Writer = errorBuffer
	if log != nil {
		stderr = io.MultiWriter(errorBuffer, log)
	}
	cmd := exec.CommandContext(ctx, version.Value.PSQL, connString, "-v", "ON_ERROR_STOP=1")
	cmd.Stdin = strings.NewReader(script)
	cmd.Stderr = stderr
	if log != nil {
		cmd.Args = append(cmd.Args, "-e")
		cmd.Stdout = log
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error running psql v%s: %s",
			version.Value.Version, strings.TrimSpace(errorBuffer.String()),
		)
	}

	return nil
}

func runPSQLCommand(
	ctx context.Context, version PGVersion, connString, sql string, log io.Writer,
) error {
	errorBuffer := &bytes.Buffer{}
	var stderr io.Writer = errorBuffer
	if log != nil {
		stderr = io.MultiWriter(errorBuffer, log)
	}
	cmd := exec.CommandContext(ctx, version.Value.PSQL, connString, "-v", "ON_ERROR_STOP=1", "-c", sql)
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"error running psql v%s: %s",
			version.Value.Version, strings.TrimSpace(errorBuffer.String()),
		)
	}

	return nil
}
