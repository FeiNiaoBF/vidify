package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

func mustRepoRootFromTestFile(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve current test file path")
	}
	return filepath.Dir(filepath.Dir(file))
}

func mustBuildFakeWhisper(t *testing.T, outPath string) {
	t.Helper()
	source := `package main

import (
	"fmt"
	"os"
)

func main() {
	var outBase string
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "--output-file" && i+1 < len(os.Args) {
			outBase = os.Args[i+1]
		}
		if os.Args[i] == "--version" {
			fmt.Println("whisper-cli fake 1.0")
			return
		}
	}
	if outBase == "" {
		fmt.Fprintln(os.Stderr, "missing --output-file")
		os.Exit(2)
	}

	srt := "1\n00:00:00,000 --> 00:00:01,000\nhello from fake whisper\n"
	if err := os.WriteFile(outBase+".srt", []byte(srt), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
`
	mustBuildInlineGoBinary(t, outPath, source)
}

func mustBuildFakeFFmpeg(t *testing.T, outPath string) {
	t.Helper()
	source := `package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		fmt.Println("ffmpeg fake 1.0")
		return
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "missing output path")
		os.Exit(2)
	}
	output := os.Args[len(os.Args)-1]
	if err := os.WriteFile(output, []byte("fake-mp4"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
`
	mustBuildInlineGoBinary(t, outPath, source)
}

func mustBuildInlineGoBinary(t *testing.T, outPath, source string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(outPath))

	tmp := t.TempDir()
	src := filepath.Join(tmp, "main.go")
	mustWriteFile(t, src, []byte(source))

	cmd := exec.Command("go", "build", "-o", outPath, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build fake binary %s failed: %v\noutput:\n%s", outPath, err, string(out))
	}
}

func mustBuildGoBinary(t *testing.T, workDir, outPath string, pkg string) {
	t.Helper()
	mustMkdirAll(t, filepath.Dir(outPath))

	cmd := exec.Command("go", "build", "-o", outPath, pkg)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build %s failed: %v\noutput:\n%s", pkg, err, string(out))
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func platformDirName() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return runtime.GOOS + "-" + arch
}

func whisperBinaryName() string {
	if runtime.GOOS == "windows" {
		return "whisper-cli.exe"
	}
	return "whisper-cli"
}

func ffmpegBinaryName() string {
	if runtime.GOOS == "windows" {
		return "ffmpeg.exe"
	}
	return "ffmpeg"
}

func exeExt() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

type noopProgress struct{}

func (n *noopProgress) Start(_ string, _ int64) {}
func (n *noopProgress) Update(_ int64)          {}
func (n *noopProgress) Finish()                 {}
func (n *noopProgress) Wait()                   {}

var _ port.ProgressReporter = (*noopProgress)(nil)

type noopLogger struct{}

func (n noopLogger) Debug(_ string, _ ...any)      {}
func (n noopLogger) Info(_ string, _ ...any)       {}
func (n noopLogger) Warn(_ string, _ ...any)       {}
func (n noopLogger) Error(_ string, _ ...any)      {}
func (n noopLogger) With(_ ...any) port.Logger     { return n }
func (n noopLogger) WithError(_ error) port.Logger { return n }

var _ port.Logger = noopLogger{}
