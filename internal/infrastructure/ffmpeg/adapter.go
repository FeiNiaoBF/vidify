// Package ffmpeg implements the port.VideoMerger driven port by shelling out
// to the ffmpeg binary via os/exec.
package ffmpeg

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

// lavfiFallback is the FFmpeg lavfi source used when no background image is
// available. It produces a 1920×1080 static frame at 1 fps in Floral White
// (米白色, #FFFAF0) — a neutral tone that works well behind dark subtitle text.
const lavfiFallback = "color=c=0xFFFAF0:s=1920x1080:r=1"

// Adapter calls ffmpeg to merge audio, subtitles, and a background into
// a final MP4 video.
type Adapter struct {
	logger port.Logger
}

// NewAdapter creates an FFmpeg adapter.
func NewAdapter(logger port.Logger) *Adapter {
	return &Adapter{logger: logger}
}

// Merge combines audio + background + SRT subtitles into a single MP4.
//
// Background strategy (in priority order):
//  1. cfg.BGImagePath is set and the file exists → use the image
//  2. Otherwise → generate a solid colour frame via FFmpeg's lavfi source
//     (no image file required; works on first run with no assets)
//
// Subtitle filter strategy:
//
//	The subtitle path is written to a temp file and passed via -filter_script
//	to avoid FFmpeg's filtergraph parser interpreting commas, colons, or
//	single quotes that may appear in legitimate file-system paths.
func (a *Adapter) Merge(ctx context.Context, audio domain.Audio, subtitle domain.Subtitle, cfg domain.Config) (domain.Video, error) {
	binPath := cfg.FFmpegBin
	if binPath == "" {
		binPath = "ffmpeg"
	}

	// Determine whether a background image is available.
	useBGImage := false
	if cfg.BGImagePath != "" {
		if _, err := os.Stat(cfg.BGImagePath); err == nil {
			useBGImage = true
		} else {
			a.logger.Warn("background image not found, using lavfi fallback",
				"path", cfg.BGImagePath,
			)
		}
	}

	// Build output file path (collision-safe).
	outputDir := cfg.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(audio.Path)
	}
	baseName := strings.TrimSuffix(filepath.Base(audio.Path), filepath.Ext(audio.Path))
	outputPath := safeOutputPath(outputDir, baseName, ".mp4")
	if outputPath != filepath.Join(outputDir, baseName+".mp4") {
		a.logger.Warn("output file already exists, writing to alternate path",
			"path", outputPath,
		)
	}

	// Write the subtitle path to a temp filter-script file so FFmpeg's
	// filtergraph parser never sees the raw path.
	filterScript, cleanup, err := buildSubtitleFilter(subtitle.SRTPath)
	if err != nil {
		return domain.Video{}, errors.Wrap(err, "build subtitle filter")
	}
	defer cleanup()

	var args []string
	if useBGImage {
		args = []string{
			"-y",
			"-loop", "1",
			"-i", cfg.BGImagePath,
			"-i", audio.Path,
			"-filter_script:v", filterScript,
			"-shortest",
			"-c:v", "libx264",
			"-tune", "stillimage",
			"-c:a", "aac",
			"-b:a", "192k",
			"-pix_fmt", "yuv420p",
			outputPath,
		}
	} else {
		// lavfi generates the video stream; audio is a separate input.
		args = []string{
			"-y",
			"-f", "lavfi", "-i", lavfiFallback,
			"-i", audio.Path,
			"-filter_script:v", filterScript,
			"-shortest",
			"-c:v", "libx264",
			"-tune", "stillimage",
			"-c:a", "aac",
			"-b:a", "192k",
			"-pix_fmt", "yuv420p",
			outputPath,
		}
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
		a.logger.WithError(err).Error("ffmpeg merge failed",
			"output", string(output),
		)
		return domain.Video{}, errors.Wrapf(domain.ErrMergeFailed,
			"ffmpeg exited with error: %v", err)
	}

	a.logger.Debug("ffmpeg output", "output", string(output))

	return domain.Video{OutputPath: outputPath}, nil
}

// buildSubtitleFilter writes a one-line FFmpeg filter-script to a temp file
// and returns the file path plus a cleanup function.
//
// Using -filter_script instead of inline -vf "subtitles=..." sidesteps
// FFmpeg's filtergraph parser, which would otherwise misinterpret commas,
// colons, and single quotes that can appear in normal file-system paths.
func buildSubtitleFilter(srtPath string) (scriptPath string, cleanup func(), err error) {
	// ffmpeg on Windows accepts forward-slash paths everywhere.
	normalized := filepath.ToSlash(srtPath)

	// Single-quotes inside the path would break the filter expression even in
	// a script file; escape them with the libass convention (\').
	escaped := strings.ReplaceAll(normalized, "'", `\'`)

	filterLine := fmt.Sprintf("subtitles='%s'", escaped)

	tmp, err := os.CreateTemp("", "vidify-filter-*.txt")
	if err != nil {
		return "", func() {}, fmt.Errorf("create filter temp file: %w", err)
	}

	if _, err := fmt.Fprintln(tmp, filterLine); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", func() {}, fmt.Errorf("write filter temp file: %w", err)
	}
	tmp.Close()

	return tmp.Name(), func() { os.Remove(tmp.Name()) }, nil
}

// safeOutputPath returns a path that does not currently exist on disk.
// If <dir>/<base><ext> is taken it tries <base>_1<ext>, <base>_2<ext>, …
// up to 999 before giving up and returning the original (overwrite is then
// left to FFmpeg's -y flag).
func safeOutputPath(dir, base, ext string) string {
	candidate := filepath.Join(dir, base+ext)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	for i := 1; i < 1000; i++ {
		candidate = filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	// Exhausted: fall back to original and let FFmpeg overwrite.
	return filepath.Join(dir, base+ext)
}
