package restorations

import (
	"fmt"
	"testing"
)

func TestRestorationLogTailKeepsLastLines(t *testing.T) {
	t.Helper()

	var saved []string
	tail := newRestorationLogTail(func(lines []string) {
		saved = append([]string(nil), lines...)
	})

	total := restorationLogTailMaxLines + 5
	for i := 1; i <= total; i++ {
		line := fmt.Sprintf("line %d\n", i)
		if _, err := tail.Write([]byte(line)); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	tail.flushNow()

	if len(saved) != restorationLogTailMaxLines {
		t.Fatalf("expected %d lines, got %d: %v", restorationLogTailMaxLines, len(saved), saved)
	}
	wantFirst := fmt.Sprintf("line %d", total-restorationLogTailMaxLines+1)
	wantLast := fmt.Sprintf("line %d", total)
	if saved[0] != wantFirst || saved[len(saved)-1] != wantLast {
		t.Fatalf("unexpected tail: %v", saved)
	}
}

func TestLogTailToLines(t *testing.T) {
	t.Helper()

	got := LogTailToLines("a\n\nb\nc\nd\ne\nf\n")
	if len(got) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(got))
	}
	if got[0] != "a" || got[len(got)-1] != "f" {
		t.Fatalf("unexpected lines: %v", got)
	}

	raw := ""
	for i := 1; i <= restorationLogTailMaxLines+5; i++ {
		raw += fmt.Sprintf("line %d\n", i)
	}
	got = LogTailToLines(raw)
	if len(got) != restorationLogTailMaxLines {
		t.Fatalf("expected %d lines, got %d", restorationLogTailMaxLines, len(got))
	}
	if got[0] != "line 6" || got[len(got)-1] != fmt.Sprintf("line %d", restorationLogTailMaxLines+5) {
		t.Fatalf("unexpected lines: %v", got)
	}
}
