package postgres

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/flate"
	_ "github.com/lib/pq"
)

const (
	parallelDumpFormat     = "pgwarden-parallel-v1"
	parallelDumpChunkSize  = 256 * 1024
	parallelDumpChunkCount = 256

	parallelDumpSoloBytes         = 256 * 1024 * 1024
	parallelDumpBatchBytes        = 256 * 1024 * 1024
	parallelDumpBatchMaxCount     = 200
	parallelDumpMaxExcludeArgsLen = 1 << 20

	manifestFileName = "manifest.json"
	preDataFileName  = "00-pre-data.sql"
	globalsFileName  = "10-data-globals.sql"
	postDataFileName = "99-post-data.sql"
)

type parallelDumpManifest struct {
	Format    string              `json:"format"`
	Snapshot  string              `json:"snapshot"`
	Jobs      int                 `json:"jobs"`
	CreatedAt string              `json:"created_at"`
	Files     []string            `json:"files"`
	Tables    map[string][]string `json:"tables"`
}

type parallelDumpJob struct {
	Patterns []string
	Exclude  []string
	FileName string
}

type parallelDumpTableInfo struct {
	Pattern string
	Bytes   int64
}

func planDataJobs(tables []parallelDumpTableInfo) []parallelDumpJob {
	jobs := make([]parallelDumpJob, 0, len(tables)/parallelDumpBatchMaxCount+8)

	name := func(pattern string) string {
		return fmt.Sprintf("20-data-%05d-%s.sql", len(jobs)+1, sanitizeEntryName(pattern))
	}

	i := 0
	for ; i < len(tables) && tables[i].Bytes >= parallelDumpSoloBytes; i++ {
		jobs = append(jobs, parallelDumpJob{
			Patterns: []string{tables[i].Pattern},
			FileName: name(tables[i].Pattern),
		})
	}

	var batch []string
	var batchBytes int64
	flush := func() {
		if len(batch) == 0 {
			return
		}
		jobs = append(jobs, parallelDumpJob{
			Patterns: batch,
			FileName: fmt.Sprintf("20-data-%05d-batch.sql", len(jobs)+1),
		})
		batch = nil
		batchBytes = 0
	}
	for ; i < len(tables); i++ {
		if len(batch) > 0 &&
			(batchBytes+tables[i].Bytes > parallelDumpBatchBytes ||
				len(batch) >= parallelDumpBatchMaxCount) {
			flush()
		}
		batch = append(batch, tables[i].Pattern)
		batchBytes += tables[i].Bytes
	}
	flush()

	return jobs
}

func (c *Client) ParallelDumpZip(
	ctx context.Context, version PGVersion, connString string,
	log io.Writer, jobs, compressionLevel int, params DumpParams,
) (io.Reader, error) {
	snap, err := prepareParallelDump(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("parallel dump: %w", err)
	}

	reader, writer := io.Pipe()

	if log != nil {
		log = &syncWriter{w: log}
	}

	go func() {
		err := c.runParallelDump(
			ctx, version, connString, log, jobs, compressionLevel, params, snap, writer,
		)
		if err != nil {
			writer.CloseWithError(err)
			return
		}
		writer.Close()
	}()

	return reader, nil
}

func (c *Client) runParallelDump(
	ctx context.Context, version PGVersion, connString string,
	log io.Writer, jobs, compressionLevel int, params DumpParams,
	snap *parallelDumpSnapshot, writer *io.PipeWriter,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer snap.release()

	snapshot := snap.id
	dataJobs := planDataJobs(snap.tables)
	if jobs > len(dataJobs)+1 {
		jobs = len(dataJobs) + 1
	}
	if jobs < 1 {
		jobs = 1
	}

	zw := zip.NewWriter(writer)
	defer zw.Close()
	zw.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, compressionLevel)
	})

	err := c.dumpSectionEntry(
		ctx, zw, preDataFileName,
		c.sectionArgsWithBin(version, c.sectionArgs(connString, snapshot, "pre-data", params, log != nil)),
		log,
	)
	if err != nil {
		return err
	}

	remainderExclude := make([]string, 0, len(snap.tables))
	for _, t := range snap.tables {
		remainderExclude = append(remainderExclude, t.Pattern)
	}

	queue := make(chan parallelDumpJob, len(dataJobs)+1)
	queue <- parallelDumpJob{Exclude: remainderExclude, FileName: globalsFileName}
	for _, j := range dataJobs {
		queue <- j
	}
	close(queue)

	entries := make(chan *rawZipEntry, len(dataJobs)+1)
	var wg sync.WaitGroup
	for range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.parallelDumpWorker(
				ctx, version, connString, snapshot, params,
				log, compressionLevel, queue, entries,
			)
		}()
	}
	go func() {
		wg.Wait()
		close(entries)
	}()

	var firstErr error
	fileNames := []string{preDataFileName}
	for e := range entries {
		if firstErr != nil {
			e.discard()
			continue
		}
		if err := writeRawEntry(zw, e); err != nil {
			firstErr = err
			cancel()
			e.discard()
			continue
		}
		fileNames = append(fileNames, e.name)
	}
	if firstErr != nil {
		return firstErr
	}

	err = c.dumpSectionEntry(
		ctx, zw, postDataFileName,
		c.sectionArgsWithBin(version, c.sectionArgs(connString, snapshot, "post-data", params, log != nil)),
		log,
	)
	if err != nil {
		return err
	}
	fileNames = append(fileNames, postDataFileName)

	tablesByFile := make(map[string][]string, len(dataJobs))
	for _, j := range dataJobs {
		tablesByFile[j.FileName] = j.Patterns
	}

	return writeParallelDumpMeta(zw, snapshot, jobs, fileNames, tablesByFile)
}

func (c *Client) parallelDumpWorker(
	ctx context.Context, version PGVersion, connString, snapshot string,
	params DumpParams, log io.Writer, compressionLevel int,
	queue <-chan parallelDumpJob, entries chan<- *rawZipEntry,
) {
	for j := range queue {
		if ctx.Err() != nil {
			return
		}

		args := c.dataArgs(connString, snapshot, j, params, log != nil)
		e := newRawZipEntry(ctx, j.FileName)
		entries <- e
		e.run(ctx, version.Value.PGDump, args, log, compressionLevel)
		if e.err != nil {
			return
		}
	}
}

type rawZipEntry struct {
	name   string
	chunks chan []byte
	done   chan struct{}
	ctx    context.Context

	err      error
	crc      uint32
	rawSize  uint64
	compSize uint64
}

func newRawZipEntry(ctx context.Context, name string) *rawZipEntry {
	return &rawZipEntry{
		name:   name,
		chunks: make(chan []byte, parallelDumpChunkCount),
		done:   make(chan struct{}),
		ctx:    ctx,
	}
}

func (e *rawZipEntry) run(
	ctx context.Context, pgDumpBin string, args []string,
	log io.Writer, compressionLevel int,
) {
	defer close(e.chunks)
	defer close(e.done)

	errTail := &tailBuffer{}
	var stderr io.Writer = errTail
	if log != nil {
		stderr = io.MultiWriter(errTail, log)
	}

	cmd := exec.CommandContext(ctx, pgDumpBin, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.err = err
		return
	}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		e.err = err
		return
	}

	crcW := crc32.NewIEEE()
	compCount := &countingChunkWriter{entry: e}
	comp, err := flate.NewWriter(compCount, compressionLevel)
	if err != nil {
		_ = cmd.Wait()
		e.err = err
		return
	}

	rawN, copyErr := io.Copy(io.MultiWriter(crcW, comp), stdout)
	closeErr := comp.Close()
	waitErr := cmd.Wait()

	switch {
	case waitErr != nil:
		e.err = fmt.Errorf("pg_dump %s: %s", e.name, errTail.String())
	case copyErr != nil:
		e.err = copyErr
	case closeErr != nil:
		e.err = closeErr
	default:
		e.crc = crcW.Sum32()
		e.rawSize = uint64(rawN)
		e.compSize = compCount.count
	}
}

func (e *rawZipEntry) discard() {
	for range e.chunks {
	}
}

type countingChunkWriter struct {
	entry *rawZipEntry
	count uint64
}

func (w *countingChunkWriter) Write(p []byte) (int, error) {
	total := len(p)
	for len(p) > 0 {
		n := min(len(p), parallelDumpChunkSize)
		chunk := make([]byte, n)
		copy(chunk, p[:n])
		select {
		case w.entry.chunks <- chunk:
		case <-w.entry.ctx.Done():
			return total - len(p), w.entry.ctx.Err()
		}
		w.count += uint64(n)
		p = p[n:]
	}
	return total, nil
}

func writeRawEntry(zw *zip.Writer, e *rawZipEntry) error {
	fh := &zip.FileHeader{
		Name:     e.name,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	fh.Flags |= 0x8

	w, err := zw.CreateRaw(fh)
	if err != nil {
		return err
	}
	for chunk := range e.chunks {
		if _, err := w.Write(chunk); err != nil {
			return err
		}
	}
	<-e.done
	if e.err != nil {
		return e.err
	}

	fh.CRC32 = e.crc
	fh.CompressedSize64 = e.compSize
	fh.UncompressedSize64 = e.rawSize
	fh.CompressedSize = sizeMirror32(e.compSize)  //nolint:staticcheck
	fh.UncompressedSize = sizeMirror32(e.rawSize) //nolint:staticcheck
	return nil
}

func sizeMirror32(v uint64) uint32 {
	if v >= math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}

func (c *Client) dumpSectionEntry(
	ctx context.Context, zw *zip.Writer, name string, args []string, log io.Writer,
) error {
	errTail := &tailBuffer{}
	var stderr io.Writer = errTail
	if log != nil {
		stderr = io.MultiWriter(errTail, log)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	w, err := zw.CreateHeader(&zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: time.Now(),
	})
	if err != nil {
		_ = cmd.Wait()
		return err
	}
	_, copyErr := io.Copy(w, stdout)
	waitErr := cmd.Wait()
	if waitErr != nil {
		return fmt.Errorf("pg_dump %s: %s", name, errTail.String())
	}
	return copyErr
}

func (c *Client) baseArgs(
	connString, snapshot string, params DumpParams, verbose bool,
) []string {
	args := []string{connString, "--snapshot=" + snapshot}
	if params.LockWaitTimeout != "" {
		args = append(args, "--lock-wait-timeout="+params.LockWaitTimeout)
	}
	if verbose {
		args = append(args, "--verbose")
	}
	return args
}

func (c *Client) sectionArgs(
	connString, snapshot, section string, params DumpParams, verbose bool,
) []string {
	args := c.baseArgs(connString, snapshot, params, verbose)
	args = append(args, "--section="+section)
	if params.Clean {
		args = append(args, "--clean")
	}
	if params.IfExists {
		args = append(args, "--if-exists")
	}
	if params.NoComments {
		args = append(args, "--no-comments")
	}
	if params.Create && section == "pre-data" {
		args = append(args, "--create")
	}
	return args
}

func (c *Client) dataArgs(
	connString, snapshot string, job parallelDumpJob, params DumpParams, verbose bool,
) []string {
	args := c.baseArgs(connString, snapshot, params, verbose)
	args = append(args, "--section=data")
	for _, p := range job.Patterns {
		args = append(args, "--table="+p)
	}
	for _, p := range job.Exclude {
		args = append(args, "--exclude-table-data="+p)
	}
	return args
}

func (c *Client) sectionArgsWithBin(version PGVersion, args []string) []string {
	return append([]string{version.Value.PGDump}, args...)
}

var ErrParallelDumpUnsupported = errors.New("parallel dump unsupported for this database")

type parallelDumpSnapshot struct {
	id      string
	tables  []parallelDumpTableInfo
	release func()
}

func prepareParallelDump(
	ctx context.Context, connString string,
) (*parallelDumpSnapshot, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	var released sync.Once
	release := func() {
		released.Do(func() {
			_ = tx.Rollback()
			_ = db.Close()
		})
	}

	fail := func(err error) (*parallelDumpSnapshot, error) {
		release()
		return nil, err
	}

	if _, err := tx.ExecContext(
		ctx, "SET idle_in_transaction_session_timeout = 0",
	); err != nil {
		return fail(err)
	}

	var snapshot string
	if err := tx.QueryRowContext(
		ctx, "SELECT pg_export_snapshot()",
	).Scan(&snapshot); err != nil {
		return fail(err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT format('%I.%I', n.nspname, c.relname), pg_table_size(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'r'
		  AND n.nspname NOT IN ('pg_catalog', 'information_schema')
		  AND n.nspname NOT LIKE 'pg\_%'
		  AND NOT EXISTS (
		    SELECT FROM pg_depend d
		    WHERE d.classid = 'pg_class'::regclass
		      AND d.objid = c.oid
		      AND d.deptype = 'e'
		  )
		ORDER BY pg_table_size(c.oid) DESC
	`)
	if err != nil {
		return fail(err)
	}
	defer rows.Close()

	var tables []parallelDumpTableInfo
	for rows.Next() {
		var t parallelDumpTableInfo
		if err := rows.Scan(&t.Pattern, &t.Bytes); err != nil {
			return fail(err)
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return fail(err)
	}

	excludeArgsLen := 0
	for _, t := range tables {
		excludeArgsLen += len("--exclude-table-data=") + len(t.Pattern) + 1
	}
	if excludeArgsLen > parallelDumpMaxExcludeArgsLen {
		return fail(fmt.Errorf(
			"%w: too many tables for the remainder job command line",
			ErrParallelDumpUnsupported,
		))
	}

	return &parallelDumpSnapshot{
		id:      snapshot,
		tables:  tables,
		release: release,
	}, nil
}

var entryNameSanitizer = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

func sanitizeEntryName(s string) string {
	s = strings.ReplaceAll(s, `"`, "")
	s = entryNameSanitizer.ReplaceAllString(s, "_")
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

func writeParallelDumpMeta(
	zw *zip.Writer, snapshot string, jobs int, files []string,
	tablesByFile map[string][]string,
) error {
	manifest := parallelDumpManifest{
		Format:    parallelDumpFormat,
		Snapshot:  snapshot,
		Jobs:      jobs,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Files:     files,
		Tables:    tablesByFile,
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	fileList := strings.Join(files, " \\\n  ")
	joinSh := "#!/bin/sh\nset -e\ncat \\\n  " + fileList + " \\\n  > dump.sql\necho \"dump.sql created\"\n"
	restoreSh := "#!/bin/sh\nset -e\nif [ -z \"$1\" ]; then\n  echo \"usage: $0 <connection-string>\" >&2\n  exit 1\nfi\ncat \\\n  " + fileList + " \\\n  | psql -v ON_ERROR_STOP=1 \"$1\"\n"
	readme := "PG Warden parallel dump (" + parallelDumpFormat + ")\n\n" +
		"This archive contains a PostgreSQL dump split into files:\n" +
		"  " + preDataFileName + "   - schema (tables, types, functions)\n" +
		"  " + globalsFileName + " - sequence values, extension data, large objects\n" +
		"  20-data-*.sql     - table data, one file per table\n" +
		"  " + postDataFileName + "   - indexes, constraints, triggers\n\n" +
		"To merge into a single classic dump file:  sh join.sh\n" +
		"To restore directly:  sh restore.sh 'postgres://user:pass@host:port/db'\n" +
		"Concatenating the .sql files in name order is equivalent to a\n" +
		"single-file pg_dump of the same database.\n"

	for _, f := range []struct{ name, content string }{
		{manifestFileName, string(manifestJSON)},
		{"join.sh", joinSh},
		{"restore.sh", restoreSh},
		{"README.txt", readme},
	} {
		w, err := zw.CreateHeader(&zip.FileHeader{
			Name:     f.name,
			Method:   zip.Deflate,
			Modified: time.Now(),
		})
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(f.content)); err != nil {
			return err
		}
	}
	return nil
}

type tailBuffer struct {
	mu  sync.Mutex
	buf []byte
}

const tailBufferMax = 8 * 1024

func (b *tailBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > tailBufferMax {
		b.buf = b.buf[len(b.buf)-tailBufferMax:]
	}
	return len(p), nil
}

func (b *tailBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.buf))
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
