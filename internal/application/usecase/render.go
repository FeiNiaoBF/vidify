package usecase

import (
	"context"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// RenderInput holds all inputs needed to produce the final video.
type RenderInput struct {
	Audio    domain.Audio
	Subtitle domain.Subtitle
	Config   domain.Config
}

// RenderOutput holds the generated video.
type RenderOutput struct {
	Video domain.Video
}

// Render drives the port.VideoMerger to assemble the final MP4 from an audio
// file, its SRT subtitles, and a background image. Keeping this separate from
// Transcribe means the rendering engine (FFmpeg today, something else tomorrow)
// can be swapped without touching the transcription pipeline.
type Render struct {
	merger   port.VideoMerger
	progress port.ProgressReporter
	logger   port.Logger
}

// NewRender creates a Render use case. All dependencies are mandatory —
// same nil-panic rationale as NewTranscribe.
func NewRender(m port.VideoMerger, p port.ProgressReporter, logger port.Logger) *Render {
	if m == nil || p == nil || logger == nil {
		panic("usecase.NewRender: all dependencies are required")
	}
	return &Render{merger: m, progress: p, logger: logger}
}

// Execute assembles the final MP4 and reports progress to the reporter.
func (uc *Render) Execute(ctx context.Context, in RenderInput) (RenderOutput, error) {
	uc.logger.Info("starting video render", "audio", in.Audio.Path)

	uc.progress.Start("Merging video", 100)

	video, err := uc.merger.Merge(ctx, in.Audio, in.Subtitle, in.Config)
	if err != nil {
		uc.progress.Finish()
		return RenderOutput{}, errors.Wrap(err, "video render failed")
	}

	uc.progress.Update(100)
	uc.progress.Finish()

	uc.logger.Info("video render complete", "output", video.OutputPath)

	return RenderOutput{Video: video}, nil
}
