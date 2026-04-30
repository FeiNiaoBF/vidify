package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FeiNiaoBF/vidify/internal/application"
	"github.com/FeiNiaoBF/vidify/internal/application/usecase"
	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/ffmpeg"
	"github.com/FeiNiaoBF/vidify/internal/infrastructure/whisper"
)

func TestPipelineIntegration_AudioToSubtitleToVideo(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	modelDir := filepath.Join(tmp, "models")
	outputDir := filepath.Join(tmp, "out")
	audioPath := filepath.Join(tmp, "sample.mp3")

	mustMkdirAll(t, modelDir)
	mustMkdirAll(t, outputDir)
	mustWriteFile(t, filepath.Join(modelDir, "ggml-base.bin"), []byte("fake-model"))
	mustWriteFile(t, audioPath, []byte("fake-audio"))

	whisperBin := filepath.Join(tmp, "bin", whisperBinaryName())
	ffmpegBin := filepath.Join(tmp, "bin", ffmpegBinaryName())
	mustBuildFakeWhisper(t, whisperBin)
	mustBuildFakeFFmpeg(t, ffmpegBin)

	logger := noopLogger{}
	progress := &noopProgress{}
	transcribeUC := usecase.NewTranscribe(whisper.NewAdapter(logger, progress), progress, logger)
	renderUC := usecase.NewRender(ffmpeg.NewAdapter(logger, progress), progress, logger)
	svc := application.NewService(transcribeUC, renderUC, logger, nil)

	cfg := domain.Config{
		InputPath:       audioPath,
		OutputDir:       outputDir,
		Model:           domain.ModelBase,
		Language:        "auto",
		WhisperBin:      whisperBin,
		WhisperModelDir: modelDir,
		FFmpegBin:       ffmpegBin,
	}

	result := svc.ProcessFile(context.Background(), cfg)
	if result.Err != nil {
		t.Fatalf("unexpected pipeline error: %v", result.Err)
	}

	if result.OutputPath == "" {
		t.Fatal("expected non-empty output path")
	}
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Fatalf("expected output video to exist: %v", err)
	}

	srtPath := filepath.Join(outputDir, "sample.srt")
	if _, err := os.Stat(srtPath); err != nil {
		t.Fatalf("expected subtitle to exist at %s: %v", srtPath, err)
	}
}
