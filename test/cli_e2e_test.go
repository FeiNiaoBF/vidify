package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIEndToEnd_AudioToSubtitleToVideo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	repoRoot := mustRepoRootFromTestFile(t)
	tmp := t.TempDir()
	modelDir := filepath.Join(tmp, "models")
	outputDir := filepath.Join(tmp, "out")
	audioPath := filepath.Join(tmp, "input.mp3")

	mustMkdirAll(t, modelDir)
	mustMkdirAll(t, outputDir)
	mustWriteFile(t, filepath.Join(modelDir, "ggml-base.bin"), []byte("fake-model"))
	mustWriteFile(t, audioPath, []byte("fake-audio"))

	vidifyBin := filepath.Join(tmp, "vidify"+exeExt())
	mustBuildGoBinary(t, repoRoot, vidifyBin, "./cmd/cli")

	platform := platformDirName()
	exeDir := filepath.Dir(vidifyBin)
	bundledWhisper := filepath.Join(exeDir, "runtime-tools", "whisper", platform, whisperBinaryName())
	bundledFFmpeg := filepath.Join(exeDir, "runtime-tools", "ffmpeg", platform, ffmpegBinaryName())
	mustBuildFakeWhisper(t, bundledWhisper)
	mustBuildFakeFFmpeg(t, bundledFFmpeg)

	cfgPath := filepath.Join(tmp, "config.yaml")
	cfgYAML := fmt.Sprintf("model: base\nwhisper-model-dir: \"%s\"\nlog:\n  to-file: false\n",
		filepath.ToSlash(modelDir),
	)
	mustWriteFile(t, cfgPath, []byte(cfgYAML))

	cmd := exec.Command(vidifyBin,
		"--config", cfgPath,
		"-i", audioPath,
		"-m", "base",
		"-o", outputDir,
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("vidify cli failed: %v\noutput:\n%s", err, string(out))
	}

	videoPath := filepath.Join(outputDir, "input.mp4")
	srtPath := filepath.Join(outputDir, "input.srt")
	if _, err := os.Stat(videoPath); err != nil {
		t.Fatalf("expected output video at %s: %v\ncli output:\n%s", videoPath, err, string(out))
	}
	if _, err := os.Stat(srtPath); err != nil {
		t.Fatalf("expected subtitle at %s: %v\ncli output:\n%s", srtPath, err, string(out))
	}
}
