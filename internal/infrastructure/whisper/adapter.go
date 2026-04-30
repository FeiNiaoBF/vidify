// Package whisper implements the port.Transcriber driven port by shelling out
// to the whisper-cli binary via os/exec.
package whisper

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// reWhisperProgress matches whisper-cli's progress lines of the form:
//
//	whisper_print_progress_callback: progress = 42 %
//
// whisper.cpp emits these to stderr during transcription.
var reWhisperProgress = regexp.MustCompile(`progress\s*=\s*(\d+)\s*%`)

// Adapter calls whisper-cli to transcribe audio into SRT subtitles.
type Adapter struct {
	logger   port.Logger
	progress port.ProgressReporter
}

// NewAdapter creates a Whisper adapter. The logger is required for
// capturing command output and diagnosing transcription failures.
func NewAdapter(logger port.Logger, progress port.ProgressReporter) *Adapter {
	return &Adapter{logger: logger, progress: progress}
}

// Transcribe runs whisper-cli against the audio file and produces an SRT file.
//
// Intermediate outputs are written to cfg.TaskCacheDir when available so they
// are cleaned up as a unit when the task finishes. On success the final SRT
// is copied to the canonical output directory; on failure the cache dir is
// retained for inspection (the service layer handles this policy).
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

	// Determine the directory to write whisper's intermediate output.
	// Prefer the task-scoped cache so the caller can clean it up atomically.
	// Fall back to the configured output directory or audio's own directory.
	workDir := cfg.TaskCacheDir
	if workDir == "" {
		workDir = cfg.OutputDir
	}
	if workDir == "" {
		workDir = filepath.Dir(audio.Path)
	}

	// Determine the final output directory where the SRT will be delivered.
	finalDir := cfg.OutputDir
	if finalDir == "" {
		finalDir = filepath.Dir(audio.Path)
	}

	// whisper-cli writes <output_file>.srt automatically when --output-srt is used.
	// Use a collision-safe base name in the final dir to avoid overwriting.
	baseName := strings.TrimSuffix(filepath.Base(audio.Path), filepath.Ext(audio.Path))
	workBase := filepath.Join(workDir, baseName) // intermediate, in cache
	finalBase := safeOutputBase(finalDir, baseName)
	finalSRT := filepath.Join(finalDir, finalBase+".srt")

	// Assemble the whisper-cli arguments.
	args := []string{
		"--model", modelFile,
		"--output-srt",               // produce SRT format
		"--output-file", workBase,    // base name for output (whisper adds .srt)
		"--file", audio.Path,         // input audio
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

	cmd := exec.CommandContext(ctx, binPath, args...)

	// Pipe stderr to parse real-time progress while capturing it for errors.
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return domain.Subtitle{}, errors.Wrap(err, "attach whisper stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return domain.Subtitle{}, errors.Wrap(err, "start whisper-cli")
	}

	stderrLines := make(chan string, 64)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			stderrLines <- scanner.Text()
		}
		close(stderrLines)
	}()

	var stderrBuf strings.Builder
	lastReported := int64(-1)

	for line := range stderrLines {
		stderrBuf.WriteString(line)
		stderrBuf.WriteByte('\n')

		// Parse "progress = N %" lines. Parsing failures are silent — we
		// simply don't update the bar, which is already better than 0→100.
		if m := reWhisperProgress.FindStringSubmatch(line); m != nil {
			if pct, err := strconv.ParseInt(m[1], 10, 64); err == nil {
				if pct > 100 {
					pct = 100
				}
				if pct != lastReported {
					a.progress.Update(pct)
					lastReported = pct
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		a.logger.WithError(err).Error("whisper transcription failed",
			"stderr", stderrBuf.String(),
		)
		return domain.Subtitle{}, errors.Wrapf(domain.ErrTranscribeFailed,
			"whisper-cli exited with error: %v", err)
	}

	a.logger.Debug("whisper stdout", "output", stdoutBuf.String())

	// Whisper writes to workBase+".srt" in the cache dir.
	cachedSRT := workBase + ".srt"
	if _, err := os.Stat(cachedSRT); os.IsNotExist(err) {
		return domain.Subtitle{}, errors.Wrapf(domain.ErrTranscribeFailed,
			"expected SRT file %q was not created by whisper", cachedSRT)
	}

	// Move the SRT from the task cache to the final output directory.
	// When workDir == finalDir the rename is a no-op path-wise; we still
	// honour finalBase for collision safety.
	if cachedSRT != finalSRT {
		if err := moveFile(cachedSRT, finalSRT); err != nil {
			return domain.Subtitle{}, errors.Wrap(err, "move SRT to output directory")
		}
	}

	detectedLang := cfg.Language
	if detectedLang == "" || detectedLang == "auto" {
		detectedLang = "auto" // TODO: parse from whisper output if needed
	}

	return domain.Subtitle{
		SRTPath:  finalSRT,
		Language: detectedLang,
	}, nil
}

// moveFile moves src to dst, falling back to a copy+delete when os.Rename
// fails across filesystem boundaries (e.g. tmpfs → ext4).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device rename failed — copy then delete.
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close destination: %w", err)
	}
	return os.Remove(src)
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
