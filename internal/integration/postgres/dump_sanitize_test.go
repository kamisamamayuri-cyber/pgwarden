package postgres

import (
	"io"
	"strings"
	"testing"
)

func TestSanitizeDumpReaderSkipsTransactionTimeout(t *testing.T) {
	t.Helper()

	in := strings.Join([]string{
		"SET statement_timeout = 0;",
		"SET lock_timeout = 0;",
		"SET transaction_timeout = 0;",
		"SET client_encoding = 'UTF8';",
		"SELECT 1;",
		"",
	}, "\n")

	out, err := io.ReadAll(sanitizeDumpReader(strings.NewReader(in)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "transaction_timeout") {
		t.Fatalf("transaction_timeout line was not filtered: %q", got)
	}
	if !strings.Contains(got, "statement_timeout") {
		t.Fatal("expected statement_timeout to remain")
	}
	if !strings.Contains(got, "SELECT 1;") {
		t.Fatal("expected dump body to remain")
	}
}

func TestSanitizeDumpReaderKeepsCopyData(t *testing.T) {
	t.Helper()

	in := strings.Join([]string{
		"SET transaction_timeout = 0;",
		"COPY public.notes (id, body) FROM stdin;",
		"1\tSET transaction_timeout = 0;",
		`\.`,
		"SET transaction_timeout = 0;",
		"SELECT 1;",
		"",
	}, "\n")

	out, err := io.ReadAll(sanitizeDumpReader(strings.NewReader(in)))
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	got := string(out)
	if !strings.Contains(got, "1\tSET transaction_timeout = 0;") {
		t.Fatalf("COPY data row was filtered: %q", got)
	}
	if strings.Count(got, "transaction_timeout") != 1 {
		t.Fatalf("SET lines outside COPY must be filtered: %q", got)
	}
	if !strings.Contains(got, `\.`) {
		t.Fatalf("COPY terminator lost: %q", got)
	}
}

func TestShouldSkipDumpLine(t *testing.T) {
	t.Helper()

	cases := []struct {
		line string
		skip bool
	}{
		{"SET transaction_timeout = 0;", true},
		{"  set transaction_timeout TO 0;", true},
		{"SET statement_timeout = 0;", false},
		{"SET idle_in_transaction_session_timeout = 0;", false},
	}

	for _, tc := range cases {
		inCopyData := false
		if shouldSkipDumpLine(tc.line, &inCopyData) != tc.skip {
			t.Fatalf("line %q: skip=%v", tc.line, tc.skip)
		}
	}

	inCopyData := true
	if shouldSkipDumpLine("SET transaction_timeout = 0;", &inCopyData) {
		t.Fatal("lines inside COPY data must not be skipped")
	}
}
