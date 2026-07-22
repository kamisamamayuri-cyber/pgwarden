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

func TestSanitizeDumpReaderStripsRoleStatements(t *testing.T) {
	in := strings.Join([]string{
		`CREATE SCHEMA egrn;`,
		`ALTER SCHEMA egrn OWNER TO landbank_super;`,
		`CREATE TABLE public.t (a int);`,
		`ALTER TABLE public.t OWNER TO landbank_super;`,
		`ALTER FUNCTION public.f(integer, text) OWNER TO landbank_super;`,
		`GRANT ALL ON SCHEMA egrn TO some_role;`,
		`REVOKE ALL ON SCHEMA public FROM PUBLIC;`,
		`ALTER DEFAULT PRIVILEGES FOR ROLE landbank_super IN SCHEMA egrn GRANT SELECT ON TABLES TO reader;`,
		`COPY public.t (a) FROM stdin;`,
		`GRANT ALL ON SCHEMA fake TO nobody;`,
		`ALTER TABLE x OWNER TO y;`,
		`\.`,
		`SELECT pg_catalog.setval('public.seq', 42, true);`,
		``,
	}, "\n")

	out, err := io.ReadAll(sanitizeDumpReader(strings.NewReader(in)))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	for _, banned := range []string{
		"OWNER TO landbank_super",
		"GRANT ALL ON SCHEMA egrn",
		"REVOKE ALL ON SCHEMA public",
		"ALTER DEFAULT PRIVILEGES FOR ROLE",
	} {
		if strings.Contains(got, banned) {
			t.Errorf("expected %q to be stripped, output:\n%s", banned, got)
		}
	}

	for _, kept := range []string{
		"CREATE SCHEMA egrn;",
		"CREATE TABLE public.t (a int);",
		"COPY public.t (a) FROM stdin;",
		"GRANT ALL ON SCHEMA fake TO nobody;",
		"ALTER TABLE x OWNER TO y;",
		"SELECT pg_catalog.setval('public.seq', 42, true);",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("expected %q to be kept, output:\n%s", kept, got)
		}
	}
}
