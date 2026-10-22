package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func resolveExecutable(path, name string) (string, error) {
	candidate := name
	if path != "" {
		candidate = path
	}
	resolved, err := exec.LookPath(candidate)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return absolute, nil
}

func resolveFFprobe(ffmpegPath string) (string, error) {
	name := "ffprobe"
	if strings.EqualFold(filepath.Ext(ffmpegPath), ".exe") {
		name += ".exe"
	}
	if resolved, err := resolveExecutable(filepath.Join(filepath.Dir(ffmpegPath), name), ""); err == nil {
		return resolved, nil
	}
	resolved, err := resolveExecutable("", name)
	if err != nil {
		return "", fmt.Errorf("find ffprobe (distributed with FFmpeg): %w", err)
	}
	return resolved, nil
}

func probeAudioDuration(ctx context.Context, ffprobePath, audioPath string) (time.Duration, error) {
	output, err := runTool(ctx, ffprobePath,
		"-v", "error", "-select_streams", "a:0",
		"-show_entries", "stream=duration:format=duration", "-of", "json", audioPath,
	)
	if err != nil {
		return 0, fmt.Errorf("read audio duration: %w", err)
	}
	return parseAudioDuration(output)
}

func parseAudioDuration(output []byte) (time.Duration, error) {
	var probe struct {
		Streams []struct {
			Duration json.RawMessage `json:"duration"`
		} `json:"streams"`
		Format struct {
			Duration json.RawMessage `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &probe); err != nil {
		return 0, fmt.Errorf("parse audio duration from ffprobe: %w", err)
	}
	if len(probe.Streams) == 0 {
		return 0, errors.New("input has no audio stream")
	}
	for _, raw := range []json.RawMessage{probe.Streams[0].Duration, probe.Format.Duration} {
		if len(raw) == 0 || string(raw) == `"N/A"` {
			continue
		}
		seconds, err := strconv.ParseFloat(strings.Trim(string(raw), `"`), 64)
		if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > float64(math.MaxInt64)/float64(time.Second) {
			continue
		}
		return time.Duration(math.Round(seconds * float64(time.Second))), nil
	}
	return 0, errors.New("ffprobe did not return a positive audio duration")
}

func resolveWhisper(path string) (string, error) {
	if path != "" {
		resolved, err := resolveExecutable(path, "")
		if err != nil {
			return "", fmt.Errorf("find whisper.cpp executable %q: %w", path, err)
		}
		return resolved, nil
	}
	for _, name := range []string{"whisper-cli", "whisper", "main"} {
		if resolved, err := resolveExecutable("", name); err == nil {
			return resolved, nil
		}
	}
	return "", errors.New("whisper.cpp executable not found; install it or pass --whisper-bin")
}

func checkFFmpegCapabilities(ctx context.Context, ffmpegPath string) error {
	filters, err := runTool(ctx, ffmpegPath, "-hide_banner", "-filters")
	if err != nil {
		return fmt.Errorf("inspect FFmpeg filters: %w", err)
	}
	if !hasToolEntry(filters, "ass") {
		return errors.New("FFmpeg was built without the ASS/libass subtitle filter")
	}
	encoders, err := runTool(ctx, ffmpegPath, "-hide_banner", "-encoders")
	if err != nil {
		return fmt.Errorf("inspect FFmpeg encoders: %w", err)
	}
	for _, encoder := range []string{"libx264", "aac"} {
		if !hasToolEntry(encoders, encoder) {
			return fmt.Errorf("FFmpeg encoder %q is unavailable", encoder)
		}
	}
	return nil
}

func hasToolEntry(output []byte, expected string) bool {
	for _, line := range strings.Split(string(output), "\n") {
		for _, field := range strings.Fields(line) {
			if field == expected {
				return true
			}
		}
	}
	return false
}

func runTool(ctx context.Context, executable string, args ...string) ([]byte, error) {
	return runToolInDir(ctx, "", executable, args...)
}

func runToolInDir(ctx context.Context, dir, executable string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
		if len(message) > 4000 {
			message = message[len(message)-4000:]
		}
		base := filepath.Base(executable)
		if ctxErr := ctx.Err(); ctxErr != nil {
			if message != "" {
				return nil, fmt.Errorf("%s: %w: %v: %s", base, ctxErr, err, message)
			}
			return nil, fmt.Errorf("%s: %w: %v", base, ctxErr, err)
		}
		if message != "" {
			return nil, fmt.Errorf("%s: %w: %s", base, err, message)
		}
		return nil, fmt.Errorf("%s: %w", base, err)
	}
	return append(stdout.Bytes(), stderr.Bytes()...), nil
}
