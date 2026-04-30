package domain

import "github.com/pkg/errors"

// Sentinel errors for the domain layer.
// Consumers should check with errors.Is() rather than direct comparison,
// because these may be wrapped with additional context at call sites.
var (
	// ErrUnsupportedFormat indicates the audio file format is not in the
	// supported set (MP3, WAV, M4A, OGG, FLAC).
	ErrUnsupportedFormat = errors.New("unsupported audio format")

	// ErrFileNotFound indicates the requested file does not exist on disk.
	ErrFileNotFound = errors.New("file not found")

	// ErrModelNotFound indicates the Whisper model file is missing.
	ErrModelNotFound = errors.New("whisper model not found")

	// ErrTranscribeFailed indicates Whisper.cpp returned a non-zero exit code.
	ErrTranscribeFailed = errors.New("transcription failed")

	// ErrMergeFailed indicates FFmpeg returned a non-zero exit code.
	ErrMergeFailed = errors.New("video merge failed")

	// ErrInvalidConfig indicates the provided configuration is not valid.
	//
	// Deprecated: prefer ErrMissingInput or ErrInvalidModel for new code;
	// this sentinel is kept so existing errors.Is checks continue to work.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrMissingInput indicates that no input path was provided.
	ErrMissingInput = errors.New("input path is required")

	// ErrInvalidModel indicates that the supplied model name is not a known
	// Whisper model size (tiny, base, small, medium, large).
	ErrInvalidModel = errors.New("invalid whisper model")

	// ErrNoFFmpeg indicates that the ffmpeg binary could not be located.
	ErrNoFFmpeg = errors.New("ffmpeg binary not found")
)
