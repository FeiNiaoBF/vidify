package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

type processOptions struct {
	modelPath   string
	outputDir   string
	ffmpegPath  string
	whisperPath string
}

func newProcessCommand() *cobra.Command {
	var options processOptions
	cmd := &cobra.Command{
		Use:   "process <audio-file>",
		Short: "Generate a subtitle video and an SRT file from audio",
		Long: `Generate a subtitle video and an SRT file from one audio file.

The input audio is normalized, transcribed locally with whisper.cpp, and rendered with FFmpeg.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("process requires exactly one audio file; see %q", "vidify process --help")
			}
			if args[0] == "" {
				return errors.New("audio file path cannot be empty")
			}
			if options.modelPath == "" {
				return errors.New("--model is required")
			}
			if options.outputDir == "" {
				return errors.New("--output-dir is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			mp4Path, srtPath, err := processAudio(cmd.Context(), args[0], options)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\nCreated %s\n", mp4Path, srtPath)
			return err
		},
	}
	cmd.Flags().StringVar(&options.modelPath, "model", "", "Local whisper.cpp model file (required)")
	cmd.Flags().StringVar(&options.outputDir, "output-dir", "", "Directory for MP4 and SRT outputs (required)")
	cmd.Flags().StringVar(&options.ffmpegPath, "ffmpeg-bin", "", "Path to the FFmpeg executable")
	cmd.Flags().StringVar(&options.whisperPath, "whisper-bin", "", "Path to the whisper.cpp executable")
	return cmd
}

func processAudio(ctx context.Context, input string, options processOptions) (mp4Path, srtPath string, err error) {
	task := newTaskState()
	var workDir, mp4Temp string
	defer func() {
		err = task.finish(err)
		if mp4Temp != "" {
			_ = os.Remove(mp4Temp)
		}
		if workDir != "" {
			_ = os.RemoveAll(workDir)
		}
	}()

	inputPath, err := absoluteRegularFile(input, "audio file")
	if err != nil {
		return "", "", err
	}
	modelPath, err := absoluteRegularFile(options.modelPath, "model file")
	if err != nil {
		return "", "", err
	}
	outputDir, err := filepath.Abs(options.outputDir)
	if err != nil {
		return "", "", fmt.Errorf("resolve output directory: %w", err)
	}

	name := filepath.Base(inputPath)
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = name
	}
	if base == "." || base == string(filepath.Separator) {
		return "", "", fmt.Errorf("cannot derive output names from audio file %q", inputPath)
	}
	mp4Path = filepath.Join(outputDir, base+".mp4")
	srtPath = filepath.Join(outputDir, base+".srt")
	if err := ensureNotExists(mp4Path, srtPath); err != nil {
		return "", "", err
	}

	ffmpegPath, err := resolveExecutable(options.ffmpegPath, "ffmpeg")
	if err != nil {
		return "", "", fmt.Errorf("find FFmpeg: %w", err)
	}
	ffprobePath, err := resolveFFprobe(ffmpegPath)
	if err != nil {
		return "", "", err
	}
	whisperPath, err := resolveWhisper(options.whisperPath)
	if err != nil {
		return "", "", err
	}
	if err := checkFFmpegCapabilities(ctx, ffmpegPath); err != nil {
		return "", "", err
	}
	audioDuration, err := probeAudioDuration(ctx, ffprobePath, inputPath)
	if err != nil {
		return "", "", err
	}
	if err := ensureDirectory(outputDir); err != nil {
		return "", "", err
	}

	if err := task.advance(ctx, taskPhaseNormalize); err != nil {
		return "", "", err
	}
	workDir, err = os.MkdirTemp("", "vidify-")
	if err != nil {
		return "", "", fmt.Errorf("create temporary work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	wavPath := filepath.Join(workDir, "normalized.wav")
	if _, err := runTool(ctx, ffmpegPath,
		"-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-i", inputPath, "-map", "0:a:0", "-vn", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", wavPath,
	); err != nil {
		return "", "", fmt.Errorf("normalize input audio: %w", err)
	}

	if err := task.advance(ctx, taskPhaseTranscribe); err != nil {
		return "", "", err
	}
	transcriptPrefix := filepath.Join(workDir, "transcript")
	if _, err := runTool(ctx, whisperPath,
		"-m", modelPath, "-f", wavPath, "-oj", "-np", "-of", transcriptPrefix,
	); err != nil {
		return "", "", fmt.Errorf("transcribe audio: %w", err)
	}

	transcriptPath := transcriptPrefix + ".json"
	segments, err := readWhisperSegments(transcriptPath)
	if err != nil {
		return "", "", err
	}
	if err := task.advance(ctx, taskPhaseLayout); err != nil {
		return "", "", err
	}
	srtData, err := renderSRT(segments)
	if err != nil {
		return "", "", err
	}
	pages, err := paginateSegments(segments, audioDuration)
	if err != nil {
		return "", "", err
	}
	assData, err := renderASS(pages)
	if err != nil {
		return "", "", err
	}
	if err := task.advance(ctx, taskPhaseRender); err != nil {
		return "", "", err
	}
	assPath := filepath.Join(workDir, "captions.ass")
	mp4Temp, err = createMP4Stage(outputDir)
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(assPath, assData, 0o600); err != nil {
		return "", "", fmt.Errorf("write ASS subtitles: %w", err)
	}
	if err := writeProvisionalFont(workDir); err != nil {
		return "", "", err
	}
	if _, err := runToolInDir(ctx, workDir, ffmpegPath,
		"-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "color=c=black:s=1920x1080:r=30",
		"-i", inputPath, "-map", "0:v:0", "-map", "1:a:0",
		"-vf", "ass=captions.ass:fontsdir=fonts", "-shortest", "-r", "30",
		"-c:v", "libx264", "-pix_fmt", "yuv420p", "-c:a", "aac", "-b:a", "64k",
		"-movflags", "+faststart", mp4Temp,
	); err != nil {
		return "", "", fmt.Errorf("render MP4: %w", err)
	}
	if err := validateMP4(ctx, ffmpegPath, mp4Temp); err != nil {
		return "", "", err
	}

	if err := task.advance(ctx, taskPhasePublish); err != nil {
		return "", "", err
	}
	if err := publishSRTNoReplace(srtData, srtPath); err != nil {
		return "", "", err
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if err := publishMP4NoReplace(mp4Temp, mp4Path); err != nil {
		return "", "", fmt.Errorf("publish MP4 failed; the valid SRT remains at %s: %w", srtPath, err)
	}
	return mp4Path, srtPath, nil
}

func absoluteRegularFile(path, label string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("%s path cannot be empty", label)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", label, err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("access %s %q: %w", label, absolute, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s %q is not a regular file", label, absolute)
	}
	return absolute, nil
}

func ensureDirectory(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("output path %q is not a directory", path)
		}
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output directory %q: %w", path, err)
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("create output directory %q: %w", path, err)
	}
	return nil
}

func ensureNotExists(paths ...string) error {
	for _, path := range paths {
		_, err := os.Lstat(path)
		if err == nil {
			return fmt.Errorf("output file already exists: %s", path)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("check output path %q: %w", path, err)
		}
	}
	return nil
}
