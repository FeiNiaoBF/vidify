// Package di is the single composition root for Vidify.
//
// Build() instantiates every concrete infrastructure adapter, assembles
// the use cases, and returns a Container. No other package imports from both
// infrastructure and application layers simultaneously — that coupling is
// intentionally concentrated here so the hexagonal architecture stays clean.
package di

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/viper"

	"github.com/FeiNiaoBF/vidify/internal/application"
	"github.com/FeiNiaoBF/vidify/internal/application/usecase"
	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/appdir"
	infraconfig "github.com/FeiNiaoBF/vidify/internal/infrastructure/config"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/ffmpeg"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/logger"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/progress"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/whisper"
)

// Container holds the fully-wired runtime objects the CLI needs to run.
// Callers interact with the Processor via the port interface, keeping the
// CLI decoupled from concrete service types.
type Container struct {
	// Config holds the resolved runtime configuration.
	Config domain.Config

	// Processor is the primary driving port entry point.
	Processor port.Processor

	// Progress is exposed so the CLI can call Wait() after processing
	// to drain all in-flight progress bar output before printing results.
	Progress *progress.Reporter

	// Logger exposes the io.Closer side of the logger so the CLI can call
	// Close() on exit to flush any buffered log entries to the rotate file.
	Logger io.Closer
}

// Build wires all infrastructure adapters, composes the use cases, creates
// the application service, and returns a ready-to-use Container.
//
// whisperBin and ffmpegBin are the resolved paths to the respective binaries.
// Path resolution is an OS/platform concern that belongs in the adapter layer,
// not inside application or domain code — so it is passed in rather than
// looked up here.
func Build(v *viper.Viper, whisperBin, ffmpegBin string) (*Container, error) {
	// Ensure all standard application directories exist before any adapter
	// tries to write to them. Non-fatal: a missing log directory is handled
	// gracefully by the logger (it falls back to console-only output).
	if err := appdir.EnsureAll(); err != nil {
		fmt.Fprintf(os.Stderr, "WARN appdir: %v\n", err)
	}

	// Inject the resolved binary paths so FromViper picks them up uniformly.
	v.Set("whisper-bin", whisperBin)
	v.Set("ffmpeg-bin", ffmpegBin)

	cfg, err := infraconfig.FromViper(v)
	if err != nil {
		return nil, err
	}

	// Inject default paths from appdir for fields the user left unconfigured.
	// We only set a default when the field is truly empty (not overridden by
	// flag, env, or config file).
	if cfg.WhisperModelDir == "" {
		if d, err := appdir.Models(); err == nil {
			cfg.WhisperModelDir = d
		}
	}
	// Only inject the default background if the file already exists on disk.
	// Asset extraction (writing the file) is Phase 2 GUI work, not MVP.
	if cfg.BGImagePath == "" {
		if p, err := appdir.DefaultBackground(); err == nil {
			if _, err := os.Stat(p); err == nil {
				cfg.BGImagePath = p
			}
		}
	}

	// Build the logger from the dedicated log.* Viper namespace so that log
	// configuration never leaks into domain.Config (which describes the task).
	// --verbose is honoured as a shortcut alias: it forces level to debug.
	logLevel := v.GetString("log.level")
	if v.GetBool("verbose") {
		logLevel = "debug"
	}
	logCfg := logger.Config{
		Dir:        v.GetString("log.dir"),
		Level:      logLevel,
		ToFile:     v.GetBool("log.to-file"),
		MaxBackups: v.GetInt("log.max-backups"),
		MaxAge:     v.GetInt("log.max-age"),
		MaxSize:    v.GetInt("log.max-size"),
	}
	log := logger.NewWithConfig(logCfg)

	whisperAdapter := whisper.NewAdapter(log)
	// ffmpeg.NewAdapter no longer takes a defaultBG — background fallback is
	// handled inside Merge() itself (lavfi colour source when no image exists).
	ffmpegAdapter := ffmpeg.NewAdapter(log)
	progressReporter := progress.NewReporter()

	transcribeUC := usecase.NewTranscribe(whisperAdapter, progressReporter, log)
	renderUC := usecase.NewRender(ffmpegAdapter, progressReporter, log)
	svc := application.NewService(transcribeUC, renderUC, log)

	// Type-assert the logger to io.Closer. The slogLogger struct satisfies
	// io.Closer (lumberjack.Logger underneath), but the port.Logger interface
	// does not expose Close() — this keeps the port clean.
	var logCloser io.Closer
	if c, ok := log.(io.Closer); ok {
		logCloser = c
	}

	return &Container{
		Config:    cfg,
		Processor: svc,
		Progress:  progressReporter,
		Logger:    logCloser,
	}, nil
}
