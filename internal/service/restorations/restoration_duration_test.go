package restorations

import (
	"database/sql"
	"testing"
	"time"
)

func TestRestorationDurationFinished(t *testing.T) {
	t.Helper()

	start := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	finish := start.Add(5*time.Minute + 12*time.Second)

	got := RestorationDuration(start, sql.NullTime{Valid: true, Time: finish})
	if got != "5m12s" {
		t.Fatalf("got %q want 5m12s", got)
	}
}

func TestRestorationDurationRunning(t *testing.T) {
	t.Helper()

	start := time.Now().Add(-90 * time.Second)
	got := RestorationDuration(start, sql.NullTime{})
	if got == "" || got == "0s" {
		t.Fatalf("expected positive elapsed duration, got %q", got)
	}
}
