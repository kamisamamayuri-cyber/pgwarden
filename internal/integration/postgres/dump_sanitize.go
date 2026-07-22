package postgres

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// pgDump SET lines for parameters that may not exist on older PostgreSQL
// versions than the pg_dump client (e.g. transaction_timeout added in PG 17).
var incompatibleDumpSetLine = regexp.MustCompile(
	`(?i)^\s*set\s+(` + strings.Join(incompatibleDumpSetParams, "|") + `)\b`,
)

var incompatibleDumpSetParams = []string{
	"transaction_timeout",
}

var copyStartLine = regexp.MustCompile(`(?i)^copy\s.*\sfrom\s+stdin;`)

// Statements referencing source-cluster roles, which may not exist on the
// restore target. Ownership is assigned by the restore flow itself (preset
// owner), so these are dropped — the restore-side equivalent of
// pg_dump --no-owner --no-acl. Matched only at column 0 with a trailing
// semicolon, the way pg_dump emits them, to avoid touching function bodies.
var roleDependentDumpLine = regexp.MustCompile(
	`(?i)^(alter\s+.*\s+owner\s+to\s+[^;]+;|grant\s+[^;]*;|revoke\s+[^;]*;|alter\s+default\s+privileges\s+[^;]*;)\s*$`,
)

// sanitizeDumpReader strips incompatible SET lines from a pg_dump SQL stream.
// Lines inside COPY ... FROM stdin; data blocks are passed through untouched:
// table data may legitimately contain lines that look like SET commands.
func sanitizeDumpReader(src io.Reader) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		br := bufio.NewReader(src)
		inCopyData := false
		for {
			line, err := br.ReadString('\n')
			if len(line) > 0 && !shouldSkipDumpLine(line, &inCopyData) {
				if _, werr := pw.Write([]byte(line)); werr != nil {
					_ = pw.CloseWithError(werr)
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					_ = pw.Close()
					return
				}
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	return pr
}

func shouldSkipDumpLine(line string, inCopyData *bool) bool {
	if *inCopyData {
		if isCopyDataTerminator(line) {
			*inCopyData = false
		}
		return false
	}

	if copyStartLine.MatchString(line) {
		*inCopyData = true
		return false
	}

	return incompatibleDumpSetLine.MatchString(line) ||
		roleDependentDumpLine.MatchString(line)
}

func isCopyDataTerminator(line string) bool {
	return strings.TrimRight(line, "\r\n") == `\.`
}
