package logtail

import (
	"strings"
	"sync"
	"time"
)

const MaxLines = 20

type Tail struct {
	mu          sync.Mutex
	lines       []string
	pending     strings.Builder
	flush       func(lines []string)
	lastFlush   time.Time
	minFlushGap time.Duration
	seq         uint64

	flushMu    sync.Mutex
	flushedSeq uint64
}

func New(flush func(lines []string)) *Tail {
	return &Tail{
		flush:       flush,
		minFlushGap: 2 * time.Second,
	}
}

func (t *Tail) Write(p []byte) (int, error) {
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
		if len(t.lines) > MaxLines {
			t.lines = t.lines[len(t.lines)-MaxLines:]
		}
	}

	t.maybeFlushLocked()
	return len(p), nil
}

func (t *Tail) maybeFlushLocked() {
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

func (t *Tail) FlushNow() {
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

func (t *Tail) runFlush(seq uint64, lines []string) {
	t.flushMu.Lock()
	defer t.flushMu.Unlock()

	if seq <= t.flushedSeq {
		return
	}
	t.flushedSeq = seq
	t.flush(lines)
}

func Join(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > MaxLines {
		lines = lines[len(lines)-MaxLines:]
	}
	return strings.Join(lines, "\n")
}

func Parse(raw string) []string {
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
	if len(result) > MaxLines {
		result = result[len(result)-MaxLines:]
	}
	return result
}
