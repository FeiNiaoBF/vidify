// Package config implements the configuration loading adapter for Vidify.
//
// It sits on the driven (right) side of the hexagon: it knows about Viper
// (an infrastructure detail) and produces a domain.Config (a domain type).
// Both the CLI and the future GUI driving adapters call FromViper after
// setting up their own flag/env bindings — they never need to know which
// Viper keys map to which Config fields.
package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/FeiNiaoBF/vidify/internal/domain"
)

// FromViper reads the current Viper state and returns a validated domain.Config.
//
// Separation of responsibilities:
//   - CLI/GUI adapter: decides which flags exist and how they bind to Viper keys.
//   - This function: translates those Viper keys into the domain type and validates
//     the result so the application core never receives a malformed Config.
func FromViper(v *viper.Viper) (domain.Config, error) {
	cfg := domain.Config{
		InputPath:       v.GetString("input"),
		OutputDir:       v.GetString("output"),
		Model:           domain.WhisperModel(v.GetString("model")),
		Language:        v.GetString("language"),
		Translate:       v.GetBool("translate"),
		BGImagePath:     v.GetString("bg-image"),
		WhisperBin:      v.GetString("whisper-bin"),
		WhisperModelDir: v.GetString("whisper-model-dir"),
		FFmpegBin:       v.GetString("ffmpeg-bin"),
		Verbose:         v.GetBool("verbose"),
	}

	if err := cfg.Validate(); err != nil {
		return domain.Config{}, errors.Wrap(err, "config validation failed")
	}

	return cfg, nil
}
