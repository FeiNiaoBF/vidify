// Package progress implements domain.ProgressReporter using mpb (multi-progress-bar).
// It provides a terminal-based progress bar for CLI users.
package progress

import (
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Reporter wraps mpb to satisfy domain.ProgressReporter.
// Each Start() call creates a new bar; Update()/Finish() operate on it.
type Reporter struct {
	container *mpb.Progress
	bar       *mpb.Bar
}

// NewReporter creates a progress reporter backed by an mpb container.
// The container manages rendering of all bars to the terminal.
func NewReporter() *Reporter {
	return &Reporter{
		container: mpb.New(mpb.WithWidth(60)),
	}
}

// Start begins a new progress bar for the named phase.
// total is the expected number of units (e.g., bytes or percentage 100).
func (r *Reporter) Start(phase string, total int64) {
	r.bar = r.container.AddBar(total,
		mpb.PrependDecorators(
			decor.Name(phase, decor.WCSyncSpaceR),
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(
			decor.Percentage(decor.WC{W: 5}),
		),
	)
}

// Update sets the bar to the given absolute value.
// We use IncrBy with the delta because mpb bars are increment-based.
func (r *Reporter) Update(current int64) {
	if r.bar != nil {
		r.bar.SetCurrent(current)
	}
}

// Finish completes the current bar and marks it done.
func (r *Reporter) Finish() {
	if r.bar != nil {
		r.bar.Abort(false) // false = complete normally, don't drop
	}
}

// Wait blocks until all bars have finished rendering.
// Call this before the program exits to ensure clean terminal output.
func (r *Reporter) Wait() {
	r.container.Wait()
}
