package progress

import (
	"io"
	"sync"
	"time"
)

type Reader struct {
	r      io.Reader
	report func(int64)
	minGap time.Duration

	mu        sync.Mutex
	total     int64
	lastFlush time.Time
}

func NewReader(r io.Reader, report func(int64)) *Reader {
	return &Reader{r: r, report: report, minGap: 2 * time.Second}
}

func (p *Reader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.mu.Lock()
		p.total += int64(n)
		total := p.total
		flush := time.Since(p.lastFlush) >= p.minGap
		if flush {
			p.lastFlush = time.Now()
		}
		p.mu.Unlock()

		if flush {
			p.report(total)
		}
	}
	return n, err
}
