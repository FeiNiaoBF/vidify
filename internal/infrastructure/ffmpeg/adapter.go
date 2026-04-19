// Package ffmpeg implements the port.VideoMerger driven port by shelling out
// to the ffmpeg binary via os/exec.
package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// Adapter calls ffmpeg to merge audio, subtitles, and a background image
// into a final MP4 video.
type Adapter struct {
	logger port.Logger

	// defaultBG is the fallback background image embedded/shipped with the app.
	// Used when the user does not supply --bg-image.
	defaultBG string
}

// NewAdapter creates an FFmpeg adapter.
// defaultBG is the path to the built-in background image (can be empty if always user-supplied).
func NewAdapter(logger port.Logger, defaultBG string) *Adapter {
	return &Adapter{
		logger:    logger,
		defaultBG: defaultBG,
	}
}

// Merge combines audio + background image + SRT subtitles into a single MP4.
//
// FFmpeg command roughly translates to:
//
//	ffmpeg -loop 1 -i bg.jpg -i audio.mp3
//	  -vf "subtitles=sub.srt" -shortest
//	  -c:v libx264 -tune stillimage -c:a aac -b:a 192k
//	  -pix_fmt yuv420p output.mp4
//
// We use -tune stillimage because the video track is a static image, which
// lets x264 compress much more efficiently than the default tune.
func (a *Adapter) Merge(ctx context.Context, audio domain.Audio, subtitle domain.Subtitle, cfg domain.Config) (domain.Video, error) {
	binPath := cfg.FFmpegBin
	if binPath == "" {
		// Fallback: assume ffmpeg is on PATH.
		binPath = "ffmpeg"
	}

	bgImage := cfg.BGImagePath
	if bgImage == "" {
		bgImage = a.defaultBG
	}
	if bgImage == "" {
		return domain.Video{}, errors.New("no background image provided and no default configured")
	}

	// Build output file path.
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(audio.Path)
	}
	baseName := strings.TrimSuffix(filepath.Base(audio.Path), filepath.Ext(audio.Path))
	outputPath := filepath.Join(outputDir, baseName+".mp4")

	// The subtitles filter path needs forward slashes and escaped colons on Windows.
	// FFmpeg's subtitle filter uses libass which expects a specific path format.
	srtFilter := formatSubtitlePath(subtitle.SRTPath)

	args := []string{
		"-y",         // overwrite output without asking
		"-loop", "1", // loop the background image forever
		"-i", bgImage, // input 0: background image
		"-i", audio.Path, // input 1: audio file
		"-vf", fmt.Sprintf("subtitles=%s", srtFilter), // burn-in subtitles
		"-shortest",       // stop when the shortest input (audio) ends
		"-c:v", "libx264", // video codec
		"-tune", "stillimage", // optimize for static image + audio
		"-c:a", "aac", // audio codec
		"-b:a", "192k", // audio bitrate — good balance of quality vs size
		"-pix_fmt", "yuv420p", // pixel format required for broad player compatibility
		outputPath,
	}

	a.logger.Info("starting ffmpeg merge",
		"bin", binPath,
		"audio", audio.Path,
		"srt", subtitle.SRTPath,
		"output", outputPath,
	)

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		a.logger.Error("ffmpeg merge failed",
			"output", string(output),
			"error", err,
		)
		return domain.Video{}, errors.Wrapf(domain.ErrMergeFailed,
			"ffmpeg exited with error: %v", err)
	}

	a.logger.Debug("ffmpeg output", "output", string(output))

	return domain.Video{
		OutputPath: outputPath,
	}, nil
}

// formatSubtitlePath escapes the SRT path for use in FFmpeg's subtitle filter.
// On Windows, libass requires colons and backslashes to be escaped.
func formatSubtitlePath(p string) string {
	// Replace backslashes with forward slashes (cross-platform safety).
	p = strings.ReplaceAll(p, `\`, `/`)
	// Escape colons (e.g., "C:" → "C\\:") for libass filter syntax.
	p = strings.ReplaceAll(p, ":", `\:`)
	return p
}
