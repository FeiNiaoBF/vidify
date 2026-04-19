// Package whisper implements the port.Transcriber driven port by shelling out
// to the whisper-cli binary via os/exec.
package whisper

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// Adapter calls whisper-cli to transcribe audio into SRT subtitles.
type Adapter struct {
	logger port.Logger
}

// NewAdapter creates a Whisper adapter. The logger is required for
// capturing command output and diagnosing transcription failures.
func NewAdapter(logger port.Logger) *Adapter {
	return &Adapter{logger: logger}
}

// Transcribe runs whisper-cli against the audio file and produces an SRT file.
// The SRT file is written next to the audio by whisper-cli, then we return its path.
func (a *Adapter) Transcribe(ctx context.Context, audio domain.Audio, cfg domain.Config) (domain.Subtitle, error) {
	binPath := cfg.WhisperBin
	if binPath == "" {
		return domain.Subtitle{}, errors.New("whisper binary path is not configured")
	}

	// Build the model file path.
	// Whisper.cpp expects "ggml-<model>.bin" in the model directory.
	modelFile := filepath.Join(cfg.WhisperModelDir, fmt.Sprintf("ggml-%s.bin", cfg.Model))
	if _, err := os.Stat(modelFile); os.IsNotExist(err) {
		return domain.Subtitle{}, errors.Wrapf(domain.ErrModelNotFound,
			"model file %q does not exist — download it first", modelFile)
	}

	// Determine output directory: prefer config output dir, fallback to audio's dir.
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(audio.Path)
	}

	// whisper-cli writes <output_file>.srt automatically when --output-srt is used.
	baseName := strings.TrimSuffix(filepath.Base(audio.Path), filepath.Ext(audio.Path))
	outputFile := filepath.Join(outputDir, baseName)

	// Assemble the whisper-cli arguments.
	args := []string{
		"--model", modelFile,
		"--output-srt",              // produce SRT format
		"--output-file", outputFile, // base name for output (whisper adds .srt)
		"--file", audio.Path, // input audio
	}

	// Language: omit for auto-detect, specify for explicit language.
	if cfg.Language != "" && cfg.Language != "auto" {
		args = append(args, "--language", cfg.Language)
	}

	if cfg.Translate {
		args = append(args, "--translate")
	}

	a.logger.Info("starting whisper transcription",
		"bin", binPath,
		"model", string(cfg.Model),
		"audio", audio.Path,
	)

	// Use CommandContext so the process is killed if the context is cancelled
	// (e.g., user presses Ctrl+C).
	cmd := exec.CommandContext(ctx, binPath, args...)

	// Capture combined output for logging; stderr has progress info,
	// stdout has the transcription result.
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Error("whisper transcription failed",
			"output", string(output),
			"error", err,
		)
		return domain.Subtitle{}, errors.Wrapf(domain.ErrTranscribeFailed,
			"whisper-cli exited with error: %v", err)
	}

	a.logger.Debug("whisper output", "output", string(output))

	srtPath := outputFile + ".srt"
	if _, err := os.Stat(srtPath); os.IsNotExist(err) {
		return domain.Subtitle{}, errors.Wrapf(domain.ErrTranscribeFailed,
			"expected SRT file %q was not created by whisper", srtPath)
	}

	detectedLang := cfg.Language
	if detectedLang == "" || detectedLang == "auto" {
		detectedLang = "auto" // TODO: parse from whisper output if needed
	}

	return domain.Subtitle{
		SRTPath:  srtPath,
		Language: detectedLang,
	}, nil
}
