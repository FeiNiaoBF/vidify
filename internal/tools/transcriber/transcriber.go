package transcriber

import (
	"context"
	"errors"
)

// ErrNotImplemented indicates the transcriber has not been fully implemented yet.
var ErrNotImplemented = errors.New("transcriber not implemented")

// Request captures the input required to produce subtitles from audio.
type Request struct {
	InputPath   string
	ModelPath   string
	Language    string
	OutputPath  string
	TempDir     string
	Environment map[string]string
}

// Result reports the generated subtitle artefact.
type Result struct {
	SubtitlePath string
}

// Service defines the contract for transcription backends.
type Service interface {
	Transcribe(ctx context.Context, req Request) (Result, error)
}

// Stub provides a placeholder implementation used during scaffolding.
type Stub struct{}

// NewStub returns a stub transcriber that surfaces ErrNotImplemented for concrete execution.
func NewStub() Stub { return Stub{} }

// Transcribe implements Service by returning ErrNotImplemented.
func (Stub) Transcribe(ctx context.Context, req Request) (Result, error) {
	return Result{}, ErrNotImplemented
}
