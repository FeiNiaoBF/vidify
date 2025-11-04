package transcriber

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CLIOptions configures the whisper CLI-backed service.
type CLIOptions struct {
	BinaryPath string
	FFmpegPath string
	Env        map[string]string
}

// CLI implements Service by shelling out to whisper-cli.
type CLI struct {
	binaryPath string
	ffmpegPath string
	baseEnv    map[string]string
}

// NewCLI constructs a CLI service ensuring the required tool paths are provided.
func NewCLI(opts CLIOptions) (*CLI, error) {
	bin := strings.TrimSpace(opts.BinaryPath)
	if bin == "" {
		return nil, errors.New("whisper binary path is required")
	}
	ffmpeg := strings.TrimSpace(opts.FFmpegPath)
	if ffmpeg == "" {
		return nil, errors.New("ffmpeg binary path is required")
	}
	return &CLI{
		binaryPath: bin,
		ffmpegPath: ffmpeg,
		baseEnv:    cloneMap(opts.Env),
	}, nil
}

// Transcribe converts the audio input to 16 kHz mono WAV via ffmpeg, invokes whisper-cli,
// and returns the generated subtitle artefact.
func (c *CLI) Transcribe(ctx context.Context, req Request) (Result, error) {
	if strings.TrimSpace(req.InputPath) == "" {
		return Result{}, errors.New("input path is required")
	}
	if strings.TrimSpace(req.ModelPath) == "" {
		return Result{}, errors.New("model path is required")
	}
	if strings.TrimSpace(req.OutputPath) == "" {
		return Result{}, errors.New("output path is required")
	}

	tempDir := strings.TrimSpace(req.TempDir)
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("prepare temp dir: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(req.OutputPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("prepare subtitle dir: %w", err)
	}

	wavFile, err := os.CreateTemp(tempDir, "vidify-*.wav")
	if err != nil {
		return Result{}, fmt.Errorf("create temp wav: %w", err)
	}
	wavPath := wavFile.Name()
	if cerr := wavFile.Close(); cerr != nil {
		return Result{}, fmt.Errorf("close temp wav: %w", cerr)
	}
	defer func() { _ = os.Remove(wavPath) }()

	convertArgs := []string{
		"-y",
		"-i", req.InputPath,
		"-ar", "16000",
		"-ac", "1",
		"-f", "wav",
		wavPath,
	}
	if err := c.runCommand(ctx, c.ffmpegPath, convertArgs, req.Environment, ""); err != nil {
		return Result{}, fmt.Errorf("ffmpeg convert: %w", err)
	}

	outputDir := filepath.Dir(req.OutputPath)
	if outputDir == "" || outputDir == "." {
		outputDir = "."
	}
	prefix := strings.TrimSuffix(filepath.Base(req.OutputPath), filepath.Ext(req.OutputPath))
	if prefix == "" {
		prefix = "output"
	}

	args := []string{
		"-m", req.ModelPath,
		"-f", wavPath,
		"-osrt",
		"-of", prefix,
	}

	if lang := strings.TrimSpace(req.Language); lang != "" {
		args = append(args, "-l", lang)
	}

	if err := c.runCommand(ctx, c.binaryPath, args, req.Environment, outputDir); err != nil {
		return Result{}, fmt.Errorf("whisper-cli: %w", err)
	}

	if _, err := os.Stat(req.OutputPath); err != nil {
		return Result{}, fmt.Errorf("subtitle missing: %w", err)
	}

	return Result{SubtitlePath: req.OutputPath}, nil
}

func (c *CLI) runCommand(ctx context.Context, bin string, args []string, overrides map[string]string, workDir string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	if strings.TrimSpace(workDir) != "" {
		cmd.Dir = workDir
	}
	cmd.Env = c.mergeEnv(overrides)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (c *CLI) mergeEnv(overrides map[string]string) []string {
	env := make(map[string]string, len(c.baseEnv)+len(overrides))
	for k, v := range sliceEnv(os.Environ()) {
		env[k] = v
	}
	for k, v := range c.baseEnv {
		env[k] = v
	}
	for k, v := range overrides {
		env[k] = v
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(env))
	for _, k := range keys {
		result = append(result, fmt.Sprintf("%s=%s", k, env[k]))
	}
	return result
}

func cloneMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func sliceEnv(items []string) map[string]string {
	env := make(map[string]string, len(items))
	for _, item := range items {
		if idx := strings.IndexByte(item, '='); idx > 0 {
			env[item[:idx]] = item[idx+1:]
		}
	}
	return env
}
