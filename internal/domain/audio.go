package domain

import (
	"fmt"
	"os"
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
// It normalises the path to absolute, validates the extension, and confirms
// the file exists on disk before returning.
func NewAudio(path string) (Audio, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Audio{}, fmt.Errorf("resolve path: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	if !supportedFormats[ext] {
		return Audio{}, ErrUnsupportedFormat
	}

	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			return Audio{}, fmt.Errorf("%w: %s", ErrFileNotFound, absPath)
		}
		return Audio{}, fmt.Errorf("stat %q: %w", absPath, err)
	}

	return Audio{
		Path:   absPath,
		Format: ext,
	}, nil
}

// IsSupportedFormat checks whether the given extension is processable.
func IsSupportedFormat(ext string) bool {
	return supportedFormats[strings.ToLower(ext)]
}
