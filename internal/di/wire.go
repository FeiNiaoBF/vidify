// Package di is the single composition root for Vidify.
//
// Build() instantiates every concrete infrastructure adapter, assembles
// the use cases, and returns a Container. No other package imports from both
// infrastructure and application layers simultaneously — that coupling is
// intentionally concentrated here so the hexagonal architecture stays clean.
package di

import (
	"github.com/spf13/viper"

	"github.com/FeiNiaoBF/vidify/internal/application"
	"github.com/FeiNiaoBF/vidify/internal/application/usecase"
	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
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
}

// Build wires all infrastructure adapters, composes the use cases, creates
// the application service, and returns a ready-to-use Container.
//
// whisperBin is the resolved path to the whisper-cli executable and is
// injected here because path resolution is an OS/platform concern that
// belongs in the adapter layer, not inside application or domain code.
func Build(v *viper.Viper, whisperBin string) (*Container, error) {
	// Inject the resolved binary path so FromViper picks it up uniformly.
	v.Set("whisper-bin", whisperBin)

	cfg, err := infraconfig.FromViper(v)
	if err != nil {
		return nil, err
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
	ffmpegAdapter := ffmpeg.NewAdapter(log, cfg.BGImagePath)
	progressReporter := progress.NewReporter()

	transcribeUC := usecase.NewTranscribe(whisperAdapter, progressReporter, log)
	renderUC := usecase.NewRender(ffmpegAdapter, progressReporter, log)
	svc := application.NewService(transcribeUC, renderUC, log)

	return &Container{
		Config:    cfg,
		Processor: svc,
		Progress:  progressReporter,
	}, nil
}
