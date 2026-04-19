// Package usecase contains the discrete application use cases for Vidify.
//
// Each use case is a single, cohesive operation: one input type, one output
// type, one driven port. This granularity makes every pipeline stage
// independently testable, replaceable, and recomposable without touching
// the application service or other use cases.
package usecase

import (
	"context"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// TranscribeInput holds all parameters needed to perform a transcription.
type TranscribeInput struct {
	Audio  domain.Audio
	Config domain.Config
}

// TranscribeOutput holds the result of a successful transcription.
type TranscribeOutput struct {
	Subtitle domain.Subtitle
}

// Transcribe drives the port.Transcriber to convert an audio file into SRT
// subtitles. Separating transcription from rendering means the two stages
// can be tested, replaced (e.g. swap Whisper for a cloud STT), or retried
// independently without touching each other or the orchestrating Service.
type Transcribe struct {
	transcriber port.Transcriber
	progress    port.ProgressReporter
	logger      port.Logger
}

// NewTranscribe creates a Transcribe use case. All dependencies are mandatory —
// panicking on nil surfaces wiring bugs at startup, not buried in a log at 2 AM.
func NewTranscribe(t port.Transcriber, p port.ProgressReporter, logger port.Logger) *Transcribe {
	if t == nil || p == nil || logger == nil {
		panic("usecase.NewTranscribe: all dependencies are required")
	}
	return &Transcribe{transcriber: t, progress: p, logger: logger}
}

// Execute performs speech-to-text on the audio file and returns the SRT result.
func (uc *Transcribe) Execute(ctx context.Context, in TranscribeInput) (TranscribeOutput, error) {
	uc.logger.Info("starting transcription",
		"audio", in.Audio.Path,
		"model", string(in.Config.Model),
	)

	uc.progress.Start("Transcribing", 100)

	subtitle, err := uc.transcriber.Transcribe(ctx, in.Audio, in.Config)
	if err != nil {
		uc.progress.Finish()
		return TranscribeOutput{}, errors.Wrap(err, "transcription failed")
	}

	uc.progress.Update(100)
	uc.progress.Finish()

	uc.logger.Info("transcription complete", "srt", subtitle.SRTPath)

	return TranscribeOutput{Subtitle: subtitle}, nil
}
