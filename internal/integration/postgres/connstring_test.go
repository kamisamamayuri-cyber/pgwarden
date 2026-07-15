package postgres

import (
	"strings"
	"testing"
)

func TestParseConnStringURI(t *testing.T) {
	t.Helper()

	info, err := ParseConnString("postgresql://pgwbackup:secret@db-staging-01:16301/myapp?sslmode=disable")
	if err != nil {
		t.Fatalf("ParseConnString: %v", err)
	}
	if info.Host != "db-staging-01" || info.Port != "16301" || info.User != "pgwbackup" {
		t.Fatalf("unexpected info: %+v", info)
	}
	if info.Database != "myapp" {
		t.Fatalf("database: %q", info.Database)
	}

	maintenance := info.WithDatabase("postgres")
	for _, part := range []string{
		"host=db-staging-01",
		"port=16301",
		"dbname=postgres",
		"user=pgwbackup",
	} {
		if !strings.Contains(maintenance, part) {
			t.Fatalf("maintenance conn %q missing %q", maintenance, part)
		}
	}
}

func TestParseConnStringKeyword(t *testing.T) {
	t.Helper()

	info, err := ParseConnString(`host=db-prod-01 port=16301 user=pgwbackup password=secret dbname=myapp sslmode=disable`)
	if err != nil {
		t.Fatalf("ParseConnString: %v", err)
	}
	if info.Database != "myapp" || info.Host != "db-prod-01" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestWithEndpoint(t *testing.T) {
	t.Helper()

	info, err := ParseConnString(`host=db-prod-01 port=16301 user=pgwbackup password=secret dbname=myapp sslmode=disable`)
	if err != nil {
		t.Fatalf("ParseConnString: %v", err)
	}

	got := info.WithEndpoint("db-rc-01", 11001, "myapp_rc")
	for _, part := range []string{
		"host=db-rc-01",
		"port=11001",
		"dbname=myapp_rc",
		"user=pgwbackup",
	} {
		if !strings.Contains(got, part) {
			t.Fatalf("conn %q missing %q", got, part)
		}
	}
}
