// Package application provides the application service that orchestrates
// the audio-to-video pipeline. It composes use cases without knowing any
// infrastructure implementation details.
package application

import (
	"context"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/application/usecase"
	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// Service is the concrete implementation of the port.Processor driving port.
// It sequences Transcribe and Render use cases for single files and batches.
// Each use case encapsulates its own driven-port interaction, so Service
// only handles: task-ID generation, directory setup, and batch iteration.
type Service struct {
	transcribe *usecase.Transcribe
	render     *usecase.Render
	logger     port.Logger
	// taskDirFn allocates a per-task working directory given a task ID.
	// It is injected from di/wire.go so the application layer never imports
	// infrastructure packages directly.
	// When nil, no task-level cache directory is used.
	taskDirFn func(taskID string) (string, error)
}

// Compile-time proof that *Service satisfies the driving port.
// A missing method becomes a build error here, not a runtime panic in production.
var _ port.Processor = (*Service)(nil)

// NewService creates an application service wired with the given use cases.
// Accepting composed use cases (not raw ports) keeps the constructor honest
// about what Service actually needs to do its job.
//
// taskDirFn is optional. When provided it allocates an isolated working
// directory for each task (used to store intermediate files and cleaned up on
// success). Pass nil to disable task-level caching (useful in tests).
func NewService(
	transcribe *usecase.Transcribe,
	render *usecase.Render,
	logger port.Logger,
	taskDirFn func(taskID string) (string, error),
) *Service {
	return &Service{
		transcribe: transcribe,
		render:     render,
		logger:     logger,
		taskDirFn:  taskDirFn,
	}
}

// ProcessFile runs the full pipeline for a single audio file:
// validate → transcribe → render → return result.
func (s *Service) ProcessFile(ctx context.Context, cfg domain.Config) port.Result {
	taskID := uuid.New().String()
	log := s.logger.With("task_id", taskID, "input", cfg.InputPath)

	log.Info("starting audio-to-video processing")

	// Validate the audio format before any I/O — cheap fail-fast.
	audio, err := domain.NewAudio(cfg.InputPath)
	if err != nil {
		log.WithError(err).Warn("invalid input audio")
		return port.Result{TaskID: taskID, InputPath: cfg.InputPath, Err: errors.Wrap(err, "invalid input audio")}
	}

	// Create the output directory early so both use cases can write there.
	if cfg.OutputDir != "" {
		if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
			log.WithError(err).Error("cannot create output directory", "dir", cfg.OutputDir)
			return port.Result{TaskID: taskID, InputPath: cfg.InputPath, Err: errors.Wrap(err, "cannot create output directory")}
		}
	}

	// Stamp the task ID so adapters can use it for per-task isolation.
	cfg.TaskID = taskID

	// Allocate a per-task cache directory if a factory function was injected.
	// On failure we log a warning and continue without a cache dir — degraded
	// but functional (adapters fall back to os.TempDir()).
	if s.taskDirFn != nil {
		cacheDir, err := s.taskDirFn(taskID)
		if err != nil {
			log.WithError(err).Warn("could not allocate task cache directory — using OS temp")
		} else {
			cfg.TaskCacheDir = cacheDir
		}
	}

	// Stage 1: Transcribe audio → SRT subtitles.
	tOut, err := s.transcribe.Execute(ctx, usecase.TranscribeInput{Audio: audio, Config: cfg})
	if err != nil {
		// Failure: retain task cache so the user can inspect intermediate files.
		if cfg.TaskCacheDir != "" {
			log.Warn("task failed — cache retained for inspection", "cache", cfg.TaskCacheDir)
		}
		return port.Result{TaskID: taskID, InputPath: cfg.InputPath, Err: err}
	}

	// Stage 2: Render audio + subtitles + background → MP4.
	rOut, err := s.render.Execute(ctx, usecase.RenderInput{Audio: audio, Subtitle: tOut.Subtitle, Config: cfg})
	if err != nil {
		// Failure: retain task cache so the user can inspect intermediate files.
		if cfg.TaskCacheDir != "" {
			log.Warn("task failed — cache retained for inspection", "cache", cfg.TaskCacheDir)
		}
		return port.Result{TaskID: taskID, InputPath: cfg.InputPath, Err: err}
	}

	// Success: remove the task cache directory to avoid unbounded disk growth.
	// The failure is logged but non-fatal — the user still gets their video.
	if cfg.TaskCacheDir != "" {
		if err := os.RemoveAll(cfg.TaskCacheDir); err != nil {
			log.WithError(err).Warn("could not remove task cache directory", "cache", cfg.TaskCacheDir)
		}
	}

	log.Info("processing complete", "output", rOut.Video.OutputPath)

	return port.Result{
		TaskID:     taskID,
		InputPath:  cfg.InputPath,
		OutputPath: rOut.Video.OutputPath,
	}
}

// ProcessBatch scans a directory for audio files and processes each one.
// It returns a result for every file found, including per-file failures.
func (s *Service) ProcessBatch(ctx context.Context, cfg domain.Config) []port.Result {
	dir := cfg.InputPath
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []port.Result{{
			TaskID:    uuid.New().String(),
			InputPath: dir,
			Err:       errors.Wrap(err, "cannot read input directory"),
		}}
	}

	var results []port.Result
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if !domain.IsSupportedFormat(ext) {
			continue
		}

		// Create a per-file config with the concrete file path.
		fileCfg := cfg
		fileCfg.InputPath = filepath.Join(dir, entry.Name())

		result := s.ProcessFile(ctx, fileCfg)
		results = append(results, result)

		// Respect context cancellation between files so the user can
		// Ctrl+C without waiting for the entire batch to complete.
		if ctx.Err() != nil {
			break
		}
	}

	if len(results) == 0 {
		return []port.Result{{
			TaskID:    uuid.New().String(),
			InputPath: dir,
			Err:       errors.New("no supported audio files found in directory"),
		}}
	}

	return results
}
