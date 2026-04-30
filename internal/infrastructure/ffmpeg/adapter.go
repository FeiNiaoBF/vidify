// Package ffmpeg implements the port.VideoMerger driven port by shelling out
// to the ffmpeg binary via os/exec.
package ffmpeg

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// lavfiFallback is the FFmpeg lavfi source used when no background image is
// available. It produces a 1920×1080 static frame at 1 fps in Floral White
// (米白色, #FFFAF0) — a neutral tone that works well behind dark subtitle text.
const lavfiFallback = "color=c=0xFFFAF0:s=1920x1080:r=1"

// reFFmpegTime matches the "time=HH:MM:SS.ss" token emitted on FFmpeg's
// progress line. FFmpeg writes this to stderr during encoding so we can
// compute elapsed-vs-total to drive a real progress bar.
var reFFmpegTime = regexp.MustCompile(`time=(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)

// reFFmpegDuration matches the "Duration: HH:MM:SS.ss" header line that
// FFmpeg prints near the beginning of its stderr output.
var reFFmpegDuration = regexp.MustCompile(`Duration:\s*(\d{2}):(\d{2}):(\d{2})\.(\d{2})`)

// Adapter calls ffmpeg to merge audio, subtitles, and a background into
// a final MP4 video.
type Adapter struct {
	logger   port.Logger
	progress port.ProgressReporter
}

// NewAdapter creates an FFmpeg adapter.
func NewAdapter(logger port.Logger, progress port.ProgressReporter) *Adapter {
	return &Adapter{logger: logger, progress: progress}
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
	// Use the task cache directory when available so cleanup is handled
	// atomically by the service; fall back to OS temp on first run / tests.
	filterScript, cleanup, err := buildSubtitleFilter(subtitle.SRTPath, cfg.TaskCacheDir)
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

	// Pipe stdout and stderr separately so we can stream-parse stderr for
	// progress while still capturing both for error reporting.
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return domain.Video{}, errors.Wrap(err, "attach ffmpeg stderr pipe")
	}

	if err := cmd.Start(); err != nil {
		return domain.Video{}, errors.Wrap(err, "start ffmpeg")
	}

	// Parse stderr in a background goroutine to update the progress bar.
	// Errors from stderr parsing are intentionally non-fatal — if the regex
	// never matches, we just don't get intermediate progress updates.
	stderrLines := make(chan string, 64)
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			stderrLines <- scanner.Text()
		}
		close(stderrLines)
	}()

	// Collect full stderr text for error messages; update progress in real time.
	var stderrBuf strings.Builder
	// throttleAt is the last progress value we reported; we only call Update
	// when the value changes by at least 1 unit to avoid redundant mpb calls.
	lastReported := int64(-1)
	// durationSecs is parsed from FFmpeg's header line "Duration: HH:MM:SS.ss".
	// It starts at 0; progress updates are skipped until we have a valid value.
	var durationSecs float64

	for line := range stderrLines {
		stderrBuf.WriteString(line)
		stderrBuf.WriteByte('\n')

		// Parse total duration from FFmpeg's header (appears once per run).
		if durationSecs == 0 {
			if d, ok := parseFFmpegDuration(line); ok {
				durationSecs = d
			}
		}

		if durationSecs <= 0 {
			continue
		}
		if elapsed, ok := parseFFmpegTime(line); ok {
			pct := int64(elapsed.Seconds() / durationSecs * 100)
			if pct > 100 {
				pct = 100
			}
			if pct != lastReported {
				a.progress.Update(pct)
				lastReported = pct
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		a.logger.WithError(err).Error("ffmpeg merge failed",
			"stderr", stderrBuf.String(),
		)
		return domain.Video{}, errors.Wrapf(domain.ErrMergeFailed,
			"ffmpeg exited with error: %v", err)
	}

	a.logger.Debug("ffmpeg stdout", "output", stdoutBuf.String())

	return domain.Video{OutputPath: outputPath}, nil
}

// parseFFmpegTime extracts the elapsed time from an FFmpeg stderr progress line
// of the form "…time=HH:MM:SS.ss…" and returns it as a time.Duration.
// The second return value is false when the line contains no such token.
func parseFFmpegTime(line string) (time.Duration, bool) {
	return parseHHMMSScc(reFFmpegTime, line)
}

// parseFFmpegDuration extracts the total duration from FFmpeg's header line
// "  Duration: HH:MM:SS.ss, …" and returns it as seconds (float64).
// The second return value is false when the line contains no such token.
func parseFFmpegDuration(line string) (float64, bool) {
	d, ok := parseHHMMSScc(reFFmpegDuration, line)
	if !ok {
		return 0, false
	}
	return d.Seconds(), true
}

// parseHHMMSScc applies re to line and converts the four capture groups
// (HH, MM, SS, cc) into a time.Duration. Returns (0, false) on no match.
func parseHHMMSScc(re *regexp.Regexp, line string) (time.Duration, bool) {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	s, _ := strconv.Atoi(m[3])
	cs, _ := strconv.Atoi(m[4]) // centiseconds

	d := time.Duration(h)*time.Hour +
		time.Duration(min)*time.Minute +
		time.Duration(s)*time.Second +
		time.Duration(cs)*10*time.Millisecond
	return d, true
}

// buildSubtitleFilter writes a one-line FFmpeg filter-script to a file and
// returns the file path plus a cleanup function.
//
// cacheDir, when non-empty, is the task-scoped cache directory; the script
// file is placed there so the service can delete the whole directory on
// success. When cacheDir is empty the file goes to OS temp (used in tests or
// when no task cache was allocated).
//
// Using -filter_script instead of inline -vf "subtitles=..." sidesteps
// FFmpeg's filtergraph parser, which would otherwise misinterpret commas,
// colons, and single quotes that can appear in normal file-system paths.
func buildSubtitleFilter(srtPath, cacheDir string) (scriptPath string, cleanup func(), err error) {
	// ffmpeg on Windows accepts forward-slash paths everywhere.
	normalized := filepath.ToSlash(srtPath)

	// Single-quotes inside the path would break the filter expression even in
	// a script file; escape them with the libass convention (\').
	escaped := strings.ReplaceAll(normalized, "'", `\'`)

	filterLine := fmt.Sprintf("subtitles='%s'", escaped)

	var tmp *os.File
	if cacheDir != "" {
		// Place the script in the task cache; no per-file defer cleanup needed
		// because the service will remove the whole cache dir on success.
		scriptPath = filepath.Join(cacheDir, "vidify-filter.txt")
		tmp, err = os.Create(scriptPath)
	} else {
		tmp, err = os.CreateTemp("", "vidify-filter-*.txt")
	}
	if err != nil {
		return "", func() {}, fmt.Errorf("create filter script: %w", err)
	}

	if _, err := fmt.Fprintln(tmp, filterLine); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", func() {}, fmt.Errorf("write filter script: %w", err)
	}
	tmp.Close()

	name := tmp.Name()
	if cacheDir != "" {
		// Cleanup is a no-op: the service removes the cache dir atomically.
		return name, func() {}, nil
	}
	return name, func() { os.Remove(name) }, nil
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
