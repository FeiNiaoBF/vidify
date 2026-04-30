package port

import (
	"context"

	"github.com/FeiNiaoBF/vidify/internal/domain"
)

// Transcriber is the driven port for speech-to-text conversion.
// Domain defines this interface; the Whisper infrastructure adapter implements it.
type Transcriber interface {
	// Transcribe converts an audio file to SRT subtitles.
	Transcribe(ctx context.Context, audio domain.Audio, cfg domain.Config) (domain.Subtitle, error)
}

// VideoMerger is the driven port for assembling the final MP4 video.
// Domain defines this interface; the FFmpeg infrastructure adapter implements it.
type VideoMerger interface {
	// Merge combines audio, subtitles, and a background image into an MP4.
	Merge(ctx context.Context, audio domain.Audio, subtitle domain.Subtitle, cfg domain.Config) (domain.Video, error)
}

// ProgressReporter is the driven port for reporting task progress.
// CLI uses an mpb bar; GUI can use a WebSocket event emitter — both implement this.
//
// Wait() is deliberately absent: blocking until all bars flush is a
// terminal-rendering concern (specific to mpb) that a GUI implementation
// would never need. Callers that require it type-assert to interface{ Wait() }.
type ProgressReporter interface {
	// Start begins a new progress phase with an expected total unit count.
	Start(phase string, total int64)

	// Update advances the current phase to the given absolute position.
	Update(current int64)

	// Finish marks the current phase as complete.
	Finish()
}

// Logger is the driven port for structured logging.
//
// Using an interface instead of *slog.Logger directly decouples all
// application-layer code from the slog package, making it easy to:
//   - Swap logging backends (slog → zerolog → zap)
//   - Control log output in tests without touching global state
//   - Add file output, JSON format, or remote shipping in infrastructure
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	// With returns a new Logger with the given key-value pairs prepended to
	// every subsequent log entry. Keys must be strings; values can be any type.
	With(args ...any) Logger

	// WithError returns a new Logger with the error attached under the "error"
	// key. It is sugar over With("error", err.Error()) for the common case of
	// logging a non-nil error alongside a message.
	WithError(err error) Logger
}
