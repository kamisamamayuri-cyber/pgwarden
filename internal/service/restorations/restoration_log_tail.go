package restorations

import (
	"strings"
	"sync"
	"time"
)

const restorationLogTailMaxLines = 20

type restorationLogTail struct {
	mu          sync.Mutex
	lines       []string
	pending     strings.Builder
	flush       func(lines []string)
	lastFlush   time.Time
	minFlushGap time.Duration
	seq         uint64

	// flushMu serializes flush callbacks; flushedSeq drops snapshots that were
	// taken before an already flushed one, so an older async flush can never
	// overwrite a newer log tail in the database.
	flushMu    sync.Mutex
	flushedSeq uint64
}

func newRestorationLogTail(flush func(lines []string)) *restorationLogTail {
	return &restorationLogTail{
		flush:       flush,
		minFlushGap: 2 * time.Second,
	}
}

func (t *restorationLogTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.pending.Write(p); err != nil {
		return 0, err
	}

	for {
		s := t.pending.String()
		idx := strings.IndexByte(s, '\n')
		if idx < 0 {
			break
		}

		line := strings.TrimSpace(s[:idx])
		t.pending.Reset()
		t.pending.WriteString(s[idx+1:])

		if line == "" {
			continue
		}

		t.lines = append(t.lines, line)
		if len(t.lines) > restorationLogTailMaxLines {
			t.lines = t.lines[len(t.lines)-restorationLogTailMaxLines:]
		}
	}

	t.maybeFlushLocked()
	return len(p), nil
}

func (t *restorationLogTail) maybeFlushLocked() {
	if t.flush == nil || len(t.lines) == 0 {
		return
	}
	if time.Since(t.lastFlush) < t.minFlushGap {
		return
	}

	t.lastFlush = time.Now()
	t.seq++
	seq := t.seq
	lines := append([]string(nil), t.lines...)
	go t.runFlush(seq, lines)
}

func (t *restorationLogTail) flushNow() {
	t.mu.Lock()
	if t.flush == nil || len(t.lines) == 0 {
		t.mu.Unlock()
		return
	}
	t.seq++
	seq := t.seq
	lines := append([]string(nil), t.lines...)
	t.mu.Unlock()

	t.runFlush(seq, lines)
}

func (t *restorationLogTail) runFlush(seq uint64, lines []string) {
	t.flushMu.Lock()
	defer t.flushMu.Unlock()

	if seq <= t.flushedSeq {
		return
	}
	t.flushedSeq = seq
	t.flush(lines)
}

func linesToLogTail(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > restorationLogTailMaxLines {
		lines = lines[len(lines)-restorationLogTailMaxLines:]
	}
	return strings.Join(lines, "\n")
}

// LogTailToLines parses stored log tail into at most restorationLogTailMaxLines lines.
func LogTailToLines(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	lines := strings.Split(raw, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	if len(result) > restorationLogTailMaxLines {
		result = result[len(result)-restorationLogTailMaxLines:]
	}
	return result
}
