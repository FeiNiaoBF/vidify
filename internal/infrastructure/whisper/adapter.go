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
		// Print an actionable, multi-line error so the user knows exactly what
		// to do without reading documentation.
		return domain.Subtitle{}, fmt.Errorf(
			"%w\n\n"+
				"  Model file : %s\n"+
				"  Models dir : %s\n"+
				"  Download   : vidify model download --model %s\n"+
				"  Manual URL : https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-%s.bin",
			domain.ErrModelNotFound,
			modelFile,
			cfg.WhisperModelDir,
			cfg.Model,
			cfg.Model,
		)
	}

	// Determine output directory: prefer config output dir, fallback to audio's dir.
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(audio.Path)
	}

	// whisper-cli writes <output_file>.srt automatically when --output-srt is used.
	// Use a collision-safe base name so batch runs never overwrite each other's SRT.
	baseName := strings.TrimSuffix(filepath.Base(audio.Path), filepath.Ext(audio.Path))
	outputBase := safeOutputBase(outputDir, baseName)
	outputFile := filepath.Join(outputDir, outputBase)

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
		a.logger.WithError(err).Error("whisper transcription failed",
			"output", string(output),
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

// safeOutputBase returns a base filename (without extension) that does not
// already exist with a .srt suffix in dir. It appends _1, _2, … on conflict.
func safeOutputBase(dir, base string) string {
	candidate := base
	for i := 1; i < 1000; i++ {
		srt := filepath.Join(dir, candidate+".srt")
		if _, err := os.Stat(srt); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s_%d", base, i)
	}
	return base
}
