package restorations

import (
	"path/filepath"
	"strings"
	"testing"
)

func repoPresetsPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "configs", "restore-presets.yaml.example")
}

func TestLoadPresetsFromRepoConfig(t *testing.T) {
	t.Helper()

	if err := LoadPresets(repoPresetsPath(t)); err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}

	preset, ok := findRestorePreset("myapp")
	if !ok {
		t.Fatal("expected preset")
	}
	if preset.Source.PbwName != "db-prod-01:5432-myapp" {
		t.Fatalf("unexpected source pbw name: %s", preset.Source.PbwName)
	}
	if len(preset.Targets) != 2 {
		t.Fatalf("expected 2 myapp targets, got %d", len(preset.Targets))
	}
	if len(getRestorePresets()) != 1 {
		t.Fatalf("expected 1 preset, got %d", len(getRestorePresets()))
	}
	if preset.Targets[0].Environment != "staging" {
		t.Fatalf("unexpected staging environment: %s", preset.Targets[0].Environment)
	}
	if preset.Targets[1].Environment != "rc" {
		t.Fatalf("unexpected rc environment: %s", preset.Targets[1].Environment)
	}

	target, ok := findRestoreTargetByEnvironment(preset, "rc")
	if !ok || target.PbwName != "db-rc-01:5432-myapp" {
		t.Fatalf("unexpected environment lookup: ok=%v target=%+v", ok, target)
	}
}

func TestRestorePostRequest(t *testing.T) {
	t.Helper()

	got := restorePostRequest("myapp")
	want := "POST /api/v1/restores/myapp/restore"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRestoreBackupDownloadRequest(t *testing.T) {
	t.Helper()

	got := restoreBackupDownloadRequest("myapp", "550e8400-e29b-41d4-a716-446655440000")
	want := "GET /api/v1/restores/myapp/backups/550e8400-e29b-41d4-a716-446655440000"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestRestoreEnvironmentTitleLatest(t *testing.T) {
	t.Helper()

	if err := LoadPresets(repoPresetsPath(t)); err != nil {
		t.Fatalf("LoadPresets: %v", err)
	}

	preset, ok := findRestorePreset("myapp")
	if !ok {
		t.Fatal("expected preset")
	}

	title := restoreEnvironmentTitleLatest(preset, preset.Targets[1])
	if !strings.Contains(title, "latest backup from prod") {
		t.Fatalf("expected latest backup in title, got %q", title)
	}
	if !strings.Contains(title, "rc") {
		t.Fatalf("expected environment in title, got %q", title)
	}
}

func TestParseRestoreFinishedAt(t *testing.T) {
	t.Helper()

	tm, err := ParseRestoreFinishedAt("2025-06-15")
	if err != nil {
		t.Fatalf("ParseRestoreFinishedAt date: %v", err)
	}
	if tm.UTC().Format("2006-01-02") != "2025-06-15" {
		t.Fatalf("unexpected date: %s", tm)
	}

	_, err = ParseRestoreFinishedAt("not-a-date")
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParsePresetsYAMLValidation(t *testing.T) {
	t.Helper()

	_, _, err := parsePresetsYAML([]byte(`presets: []`))
	if err == nil {
		t.Fatal("expected empty presets error")
	}

	_, _, err = parsePresetsYAML([]byte(`
presets:
  - id: one
    title: one
    source: { host: a, port: 5432, database: db1 }
    targets:
      - environment: stage
        host: b
        port: 5432
        database: db2
      - environment: stage
        host: c
        port: 5432
        database: db3
`))
	if err == nil {
		t.Fatal("expected duplicate environment error")
	}
}
