package renderer

import (
	"context"
	"errors"
)

// ErrNotImplemented indicates the renderer has not been wired to ffmpeg yet.
var ErrNotImplemented = errors.New("renderer not implemented")

// Request describes the assets required to produce a captioned video.
type Request struct {
	AudioPath    string
	SubtitlePath string
	OutputPath   string
	TempDir      string
	FFmpegPath   string
	ExtraArgs    []string
}

// Result captures metadata about the rendered video artefact.
type Result struct {
	OutputPath string
}

// Service defines the contract for video renderers.
type Service interface {
	Render(ctx context.Context, req Request) (Result, error)
}

// Stub provides a placeholder implementation used during scaffolding.
type Stub struct{}

// NewStub returns a stub renderer producing ErrNotImplemented until integrations land.
func NewStub() Stub { return Stub{} }

// Render implements Service by returning ErrNotImplemented.
func (Stub) Render(ctx context.Context, req Request) (Result, error) {
	return Result{}, ErrNotImplemented
}
