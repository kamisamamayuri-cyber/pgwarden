package logtail

import (
	"fmt"
	"testing"
)

func TestTailKeepsLastLines(t *testing.T) {
	t.Helper()

	var saved []string
	tail := New(func(lines []string) {
		saved = append([]string(nil), lines...)
	})

	total := MaxLines + 5
	for i := 1; i <= total; i++ {
		line := fmt.Sprintf("line %d\n", i)
		if _, err := tail.Write([]byte(line)); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	tail.FlushNow()

	if len(saved) != MaxLines {
		t.Fatalf("expected %d lines, got %d: %v", MaxLines, len(saved), saved)
	}
	wantFirst := fmt.Sprintf("line %d", total-MaxLines+1)
	wantLast := fmt.Sprintf("line %d", total)
	if saved[0] != wantFirst || saved[len(saved)-1] != wantLast {
		t.Fatalf("unexpected tail: %v", saved)
	}
}

func TestParse(t *testing.T) {
	t.Helper()

	got := Parse("a\n\nb\nc\nd\ne\nf\n")
	if len(got) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(got))
	}
	if got[0] != "a" || got[len(got)-1] != "f" {
		t.Fatalf("unexpected lines: %v", got)
	}

	raw := ""
	for i := 1; i <= MaxLines+5; i++ {
		raw += fmt.Sprintf("line %d\n", i)
	}
	got = Parse(raw)
	if len(got) != MaxLines {
		t.Fatalf("expected %d lines, got %d", MaxLines, len(got))
	}
	if got[0] != "line 6" || got[len(got)-1] != fmt.Sprintf("line %d", MaxLines+5) {
		t.Fatalf("unexpected lines: %v", got)
	}
}
