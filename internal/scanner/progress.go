package scanner

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Progress tracks and displays scan progress
type Progress struct {
	mu          sync.Mutex
	total       int
	completed   int
	failed      int
	startTime   time.Time
	writer      io.Writer
	enabled     bool
	lastUpdate  time.Time
	updateEvery time.Duration
}

// NewProgress creates a new progress tracker
func NewProgress(total int, writer io.Writer, enabled bool) *Progress {
	return &Progress{
		total:       total,
		completed:   0,
		failed:      0,
		startTime:   time.Now(),
		writer:      writer,
		enabled:     enabled,
		updateEvery: 500 * time.Millisecond,
	}
}

// Increment increments the completed count
func (p *Progress) Increment(success bool) {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if success {
		p.completed++
	} else {
		p.failed++
	}

	// Only update display periodically to avoid too much output
	now := time.Now()
	if now.Sub(p.lastUpdate) >= p.updateEvery {
		p.display()
		p.lastUpdate = now
	}
}

// Done marks progress as complete and displays final stats
func (p *Progress) Done() {
	if !p.enabled {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.display()
	fmt.Fprintln(p.writer)
}

// display shows current progress (must be called with lock held)
func (p *Progress) display() {
	if !p.enabled || p.writer == nil {
		return
	}

	elapsed := time.Since(p.startTime)
	total := p.completed + p.failed
	rate := float64(total) / elapsed.Seconds()

	var eta time.Duration
	if rate > 0 && p.total > 0 {
		remaining := p.total - total
		eta = time.Duration(float64(remaining)/rate) * time.Second
	}

	percent := 0
	if p.total > 0 {
		percent = (total * 100) / p.total
	}

	status := fmt.Sprintf("\r[%3d%%] %d/%d IPs | %d failed | %.1f IP/s | ETA: %s",
		percent,
		p.completed,
		p.total,
		p.failed,
		rate,
		eta.Round(time.Second),
	)

	fmt.Fprint(p.writer, status)
}

// Stats returns current statistics
func (p *Progress) Stats() (completed, failed int, elapsed time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.completed, p.failed, time.Since(p.startTime)
}
