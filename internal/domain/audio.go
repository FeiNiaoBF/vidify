package domain

import (
	"path/filepath"
	"strings"
)

// supportedFormats lists audio extensions the system can process.
// Using a map for O(1) lookup instead of a slice scan.
var supportedFormats = map[string]bool{
	".mp3":  true,
	".wav":  true,
	".m4a":  true,
	".ogg":  true,
	".flac": true,
}

// Audio represents an input audio file to be processed.
// It is a value object — immutable after creation.
type Audio struct {
	// Path is the absolute filesystem path to the audio file.
	Path string

	// Format is the lowercase extension including the dot (e.g. ".mp3").
	Format string
}

// NewAudio creates an Audio value from a file path.
// It validates that the extension is in the supported set.
func NewAudio(path string) (Audio, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if !supportedFormats[ext] {
		return Audio{}, ErrUnsupportedFormat
	}
	return Audio{
		Path:   path,
		Format: ext,
	}, nil
}

// IsSupportedFormat checks whether the given extension is processable.
func IsSupportedFormat(ext string) bool {
	return supportedFormats[strings.ToLower(ext)]
}
