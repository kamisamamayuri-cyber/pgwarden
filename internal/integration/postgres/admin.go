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

	sql := fmt.Sprintf(`
ALTER DATABASE %s OWNER TO %s;
ALTER SCHEMA public OWNER TO %s;
DO $pbw$
DECLARE
  obj record;
  target_owner text := %s;
BEGIN
  FOR obj IN
    SELECT n.nspname AS schema_name
    FROM pg_namespace n
    WHERE %s
      AND pg_get_userbyid(n.nspowner) IS NOT NULL
      AND pg_get_userbyid(n.nspowner) <> target_owner
    ORDER BY n.nspname
  LOOP
    EXECUTE format('ALTER SCHEMA %%I OWNER TO %%I', obj.schema_name, target_owner);
  END LOOP;

  FOR obj IN
    SELECT n.nspname AS schema_name, c.relname AS object_name, c.relkind
    FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE %s
      AND pg_get_userbyid(c.relowner) IS NOT NULL
      AND pg_get_userbyid(c.relowner) <> target_owner
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
  LOOP
    CASE obj.relkind
      WHEN 'r', 'p' THEN
        EXECUTE format(
          'ALTER TABLE %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'v' THEN
        EXECUTE format(
          'ALTER VIEW %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'm' THEN
        EXECUTE format(
          'ALTER MATERIALIZED VIEW %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'S' THEN
        EXECUTE format(
          'ALTER SEQUENCE %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'f' THEN
        EXECUTE format(
          'ALTER FOREIGN TABLE %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'i' THEN
        EXECUTE format(
          'ALTER INDEX %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
      WHEN 'c' THEN
        EXECUTE format(
          'ALTER TYPE %%I.%%I OWNER TO %%I',
          obj.schema_name, obj.object_name, target_owner
        );
    END CASE;
  END LOOP;

  FOR obj IN
    SELECT
      n.nspname AS schema_name,
      t.typname AS object_name
    FROM pg_type t
    JOIN pg_namespace n ON n.oid = t.typnamespace
    WHERE t.typtype = 'd'
      AND %s
      AND pg_get_userbyid(t.typowner) IS NOT NULL
      AND pg_get_userbyid(t.typowner) <> target_owner
    ORDER BY n.nspname, t.typname
  LOOP
    EXECUTE format(
      'ALTER DOMAIN %%I.%%I OWNER TO %%I',
      obj.schema_name, obj.object_name, target_owner
    );
  END LOOP;

  FOR obj IN
    SELECT
      n.nspname AS schema_name,
      p.proname AS object_name,
      pg_get_function_identity_arguments(p.oid) AS args,
      p.prokind
    FROM pg_proc p
    JOIN pg_namespace n ON n.oid = p.pronamespace
    WHERE %s
      AND pg_get_userbyid(p.proowner) IS NOT NULL
      AND pg_get_userbyid(p.proowner) <> target_owner
    ORDER BY n.nspname, p.proname, args
  LOOP
    CASE obj.prokind
      WHEN 'p' THEN
        EXECUTE format(
          'ALTER PROCEDURE %%I.%%I(%%s) OWNER TO %%I',
          obj.schema_name, obj.object_name, obj.args, target_owner
        );
      WHEN 'a' THEN
        EXECUTE format(
          'ALTER AGGREGATE %%I.%%I(%%s) OWNER TO %%I',
          obj.schema_name, obj.object_name, obj.args, target_owner
        );
      ELSE
        EXECUTE format(
          'ALTER FUNCTION %%I.%%I(%%s) OWNER TO %%I',
          obj.schema_name, obj.object_name, obj.args, target_owner
        );
    END CASE;
  END LOOP;
END
$pbw$;
`, quotedDB, quotedOwner, quotedOwner, pq.QuoteLiteral(owner),
		ownerReassignSystemNamespaceFilter,
		ownerReassignSystemNamespaceFilter,
		ownerReassignSystemNamespaceFilter,
		ownerReassignSystemNamespaceFilter)

	return runPSQLCommand(ctx, version, connString, sql, log)
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
