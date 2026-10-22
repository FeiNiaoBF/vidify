package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func validateMP4(ctx context.Context, ffmpegPath, path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("inspect rendered MP4: %w", err)
	}
	if info.Size() == 0 {
		return errors.New("rendered MP4 is empty")
	}
	if _, err := runTool(ctx, ffmpegPath, "-hide_banner", "-nostdin", "-v", "error", "-i", path, "-map", "0:v:0", "-map", "0:a:0", "-f", "null", "-"); err != nil {
		return fmt.Errorf("validate rendered MP4: %w", err)
	}
	return nil
}

func createMP4Stage(outputDir string) (string, error) {
	file, err := os.CreateTemp(outputDir, ".vidify-*.mp4")
	if err != nil {
		return "", fmt.Errorf("create MP4 staging file in output directory: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("close MP4 staging file: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("prepare MP4 staging path: %w", err)
	}
	return path, nil
}

func publishMP4NoReplace(source, destination string) error {
	if err := os.Link(source, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output file already exists: %s", destination)
		}
		return fmt.Errorf("publish staged MP4 %q: %w", source, err)
	}
	return nil
}

func publishSRTNoReplace(data []byte, destination string) error {
	if len(data) == 0 {
		return errors.New("cannot publish an empty SRT")
	}
	staged, stagedPath, err := createSRTStage(filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer os.Remove(stagedPath)

	written, writeErr := staged.Write(data)
	closeErr := staged.Close()
	if writeErr != nil {
		return fmt.Errorf("write SRT staging file: %w", errors.Join(writeErr, closeErr))
	}
	if written != len(data) {
		return fmt.Errorf("write SRT staging file: %w", errors.Join(io.ErrShortWrite, closeErr))
	}
	if closeErr != nil {
		return fmt.Errorf("close SRT staging file: %w", closeErr)
	}

	if err := os.Link(stagedPath, destination); err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("output file already exists: %s", destination)
		}
		return fmt.Errorf("publish staged SRT %q: %w", stagedPath, err)
	}
	return nil
}

func createSRTStage(outputDir string) (*os.File, string, error) {
	placeholder, err := os.CreateTemp(outputDir, ".vidify-*.srt")
	if err != nil {
		return nil, "", fmt.Errorf("create SRT staging path: %w", err)
	}
	path := placeholder.Name()
	if err := placeholder.Close(); err != nil {
		os.Remove(path)
		return nil, "", fmt.Errorf("close SRT staging placeholder: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return nil, "", fmt.Errorf("prepare SRT staging path: %w", err)
	}

	// Reopen exclusively with the normal output mode so the process umask applies.
	staged, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, "", fmt.Errorf("open SRT staging file: %w", err)
	}
	return staged, path, nil
}
