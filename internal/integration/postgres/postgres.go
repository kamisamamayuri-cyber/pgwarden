package postgres

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"

	"github.com/klauspost/compress/flate"
	"github.com/orsinium-labs/enum"
)

/*
	Important:
	Versions supported by PG Warden must be supported in PostgreSQL Version Policy
	https://www.postgresql.org/support/versioning/

	Backing up a database from an old unsupported version should not be allowed.
*/

type version struct {
	Version string
	PGDump  string
	PSQL    string
}

type PGVersion enum.Member[version]

var (
	PG13 = PGVersion{version{
		Version: "13",
		PGDump:  "/usr/lib/postgresql/13/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/13/bin/psql",
	}}
	PG14 = PGVersion{version{
		Version: "14",
		PGDump:  "/usr/lib/postgresql/14/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/14/bin/psql",
	}}
	PG15 = PGVersion{version{
		Version: "15",
		PGDump:  "/usr/lib/postgresql/15/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/15/bin/psql",
	}}
	PG16 = PGVersion{version{
		Version: "16",
		PGDump:  "/usr/lib/postgresql/16/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/16/bin/psql",
	}}
	PG17 = PGVersion{version{
		Version: "17",
		PGDump:  "/usr/lib/postgresql/17/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/17/bin/psql",
	}}
	PG18 = PGVersion{version{
		Version: "18",
		PGDump:  "/usr/lib/postgresql/18/bin/pg_dump",
		PSQL:    "/usr/lib/postgresql/18/bin/psql",
	}}

	PGVersions     = []PGVersion{PG13, PG14, PG15, PG16, PG17, PG18}
	PGVersionsDesc = []PGVersion{PG18, PG17, PG16, PG15, PG14, PG13}
)

type Client struct{}

func New() *Client {
	return &Client{}
}

// ParseVersion returns the PGVersion enum member for the given PostgreSQL
// version as a string.
func (Client) ParseVersion(version string) (PGVersion, error) {
	switch version {
	case "13":
		return PG13, nil
	case "14":
		return PG14, nil
	case "15":
		return PG15, nil
	case "16":
		return PG16, nil
	case "17":
		return PG17, nil
	case "18":
		return PG18, nil
	default:
		return PGVersion{}, fmt.Errorf("pg version not allowed: %s", version)
	}
}

// Test tests the connection to the PostgreSQL database
func (Client) Test(ctx context.Context, version PGVersion, connString string) error {
	info, err := ParseConnString(connString)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}
	if _, ok := info.Extra["connect_timeout"]; !ok {
		info.Extra["connect_timeout"] = "30"
	}
	cmd := exec.CommandContext(ctx, version.Value.PSQL, info.String(), "-c", "SELECT 1;")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"error running psql test v%s: %s",
			version.Value.Version, output,
		)
	}

	return nil
}

// DumpParams contains the parameters for the pg_dump command
type DumpParams struct {
	// DataOnly (--data-only): Dump only the data, not the schema (data definitions).
	// Table data, large objects, and sequence values are dumped.
	DataOnly bool

	// SchemaOnly (--schema-only): Dump only the object definitions (schema), not data.
	SchemaOnly bool

	// Clean (--clean): Output commands to DROP all the dumped database objects
	// prior to outputting the commands for creating them. This option is useful
	// when the restore is to overwrite an existing database. If any of the
	// objects do not exist in the destination database, ignorable error messages
	// will be reported during restore, unless --if-exists is also specified.
	Clean bool

	// IfExists (--if-exists): Use DROP ... IF EXISTS commands to drop objects in
	// --clean mode. This suppresses “does not exist” errors that might otherwise
	// be reported. This option is not valid unless --clean is also specified.
	IfExists bool

	// Create (--create): Begin the output with a command to create the database
	// itself and reconnect to the created database. (With a script of this form,
	// it doesn't matter which database in the destination installation you
	// connect to before running the script.) If --clean is also specified, the
	// script drops and recreates the target database before reconnecting to it.
	Create bool

	// NoComments (--no-comments): Do not dump comments.
	NoComments bool

	// LockWaitTimeout (--lock-wait-timeout): Fail if unable to acquire a table
	// lock within the specified timeout instead of waiting indefinitely.
	// Example value: "10min". Empty string disables the flag.
	LockWaitTimeout string

	// SerializableDeferrable (--serializable-deferrable): Run the dump in a
	// SERIALIZABLE DEFERRABLE transaction. PostgreSQL defers the transaction
	// start until a clean snapshot is available, so the dump acquires no table
	// locks and does not block concurrent writes. Recommended for primaries
	// without a streaming replica where AccessShareLock contention is a concern.
	SerializableDeferrable bool

	// CompressionLevel is the DEFLATE level for the dump zip (1-9).
	// Zero falls back to defaultDumpCompressionLevel.
	CompressionLevel int
}

const defaultDumpCompressionLevel = 3

// Dump runs the pg_dump command with the given parameters. It returns the SQL
// dump as an io.Reader. Cancelling ctx kills the pg_dump process.
func (Client) Dump(
	ctx context.Context, version PGVersion, connString string, log io.Writer, params ...DumpParams,
) io.Reader {
	pickedParams := DumpParams{}
	if len(params) > 0 {
		pickedParams = params[0]
	}

	args := []string{connString}
	if pickedParams.DataOnly {
		args = append(args, "--data-only")
	}
	if pickedParams.SchemaOnly {
		args = append(args, "--schema-only")
	}
	if pickedParams.Clean {
		args = append(args, "--clean")
	}
	if pickedParams.IfExists {
		args = append(args, "--if-exists")
	}
	if pickedParams.Create {
		args = append(args, "--create")
	}
	if pickedParams.NoComments {
		args = append(args, "--no-comments")
	}
	if pickedParams.LockWaitTimeout != "" {
		args = append(args, "--lock-wait-timeout="+pickedParams.LockWaitTimeout)
	}
	if pickedParams.SerializableDeferrable {
		args = append(args, "--serializable-deferrable")
	}
	if log != nil {
		args = append(args, "--verbose")
	}

	errorBuffer := &bytes.Buffer{}
	var stderr io.Writer = errorBuffer
	if log != nil {
		stderr = io.MultiWriter(errorBuffer, log)
	}

	reader, writer := io.Pipe()
	cmd := exec.CommandContext(ctx, version.Value.PGDump, args...)
	cmd.Stdout = writer
	cmd.Stderr = stderr

	go func() {
		defer writer.Close()
		if err := cmd.Run(); err != nil {
			writer.CloseWithError(fmt.Errorf(
				"error running pg_dump v%s: %s",
				version.Value.Version, errorBuffer.String(),
			))
		}
	}()

	return reader
}

// DumpZip runs the pg_dump command with the given parameters and returns the
// ZIP-compressed SQL dump as an io.Reader.
func (c *Client) DumpZip(
	ctx context.Context, version PGVersion, connString string, log io.Writer, params ...DumpParams,
) io.Reader {
	level := defaultDumpCompressionLevel
	if len(params) > 0 && params[0].CompressionLevel != 0 {
		level = params[0].CompressionLevel
	}

	dumpReader := c.Dump(ctx, version, connString, log, params...)
	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()

		zipWriter := zip.NewWriter(writer)
		defer zipWriter.Close()

		zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, level)
		})

		fileWriter, err := zipWriter.Create("dump.sql")
		if err != nil {
			writer.CloseWithError(fmt.Errorf("error creating zip file: %w", err))
			return
		}

		if _, err := io.Copy(fileWriter, dumpReader); err != nil {
			writer.CloseWithError(fmt.Errorf("error writing to zip file: %w", err))
			return
		}
	}()

	return reader
}
