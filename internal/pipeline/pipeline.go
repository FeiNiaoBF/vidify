package pipeline

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/FeiNiaoBF/vidify/internal/logging"
	"github.com/FeiNiaoBF/vidify/internal/tools/renderer"
	"github.com/FeiNiaoBF/vidify/internal/tools/transcriber"
)

var (
	// ErrNotImplemented signals that the pipeline cannot execute concrete work yet.
	ErrNotImplemented = errors.New("pipeline execution not implemented")
	// ErrInputRequired indicates a missing input path for a conversion job.
	ErrInputRequired = errors.New("input audio file is required")
	// ErrDirectoryRequired indicates a missing directory for batch processing.
	ErrDirectoryRequired = errors.New("input directory is required")
)

// Config controls directory layout and external tool defaults.
type Config struct {
	OutputDir string
	TempDir   string
	ModelsDir string
}

// Dependencies gathers collaborators required by the pipeline.
type Dependencies struct {
	Transcriber transcriber.Service
	Renderer    renderer.Service
	Logger      logging.Logger
}

// Pipeline orchestrates the conversion workflow.
type Pipeline struct {
	cfg         Config
	transcriber transcriber.Service
	renderer    renderer.Service
	logger      logging.Logger
}

// New constructs a pipeline instance.
func New(cfg Config, deps Dependencies) *Pipeline {
	if strings.TrimSpace(cfg.OutputDir) == "" {
		cfg.OutputDir = "output"
	}
	if strings.TrimSpace(cfg.TempDir) == "" {
		cfg.TempDir = "temp"
	}
	if strings.TrimSpace(cfg.ModelsDir) == "" {
		cfg.ModelsDir = "models"
	}

	return &Pipeline{
		cfg:         cfg,
		transcriber: deps.Transcriber,
		renderer:    deps.Renderer,
		logger:      deps.Logger,
	}
}

// ConvertOptions describes a single CLI conversion request.
type ConvertOptions struct {
	Input    string
	Output   string
	Subtitle string

	Language string
	Model    string

	DryRun bool
}

// ConvertResult reports the planned or completed conversion artefacts.
type ConvertResult struct {
	OutputPath   string
	SubtitlePath string
	Steps        []string
	DryRun       bool
}

// Convert validates the input and returns a plan for the conversion. When DryRun is false,
// it attempts to call the tool services and surfaces ErrNotImplemented until those land.
func (p *Pipeline) Convert(ctx context.Context, opts ConvertOptions) (ConvertResult, error) {
	if strings.TrimSpace(opts.Input) == "" {
		return ConvertResult{}, ErrInputRequired
	}

	// Use input filename to derive output if not provided
	output := strings.TrimSpace(opts.Output)
	if output == "" {
		base := strings.TrimSuffix(filepath.Base(opts.Input), filepath.Ext(opts.Input))
		if base == "" {
			base = "output"
		}
		output = filepath.Join(p.cfg.OutputDir, base+".mp4")
	}

	steps := p.planSteps(opts)
	baseName := strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))
	if baseName == "" {
		baseName = "output"
	}

	subtitlePath := strings.TrimSpace(opts.Subtitle)
	subtitleProvided := subtitlePath != ""
	if !subtitleProvided {
		subtitlePath = filepath.Join(filepath.Dir(output), baseName+".srt")
	}
	result := ConvertResult{OutputPath: output, SubtitlePath: subtitlePath, Steps: steps, DryRun: opts.DryRun}

	if opts.DryRun {
		p.debug("convert dry-run", logging.Field{Key: "input", Value: opts.Input}, logging.Field{Key: "output", Value: output})
		return result, nil
	}

	if subtitleProvided {
		if err := p.ensureSubtitleProvided(subtitlePath); err != nil {
			return result, err
		}
		p.info("using provided subtitle", logging.Field{Key: "subtitle", Value: subtitlePath})
	} else {
		if p.transcriber == nil {
			return result, ErrNotImplemented
		}
		transReq := transcriber.Request{
			InputPath:  opts.Input,
			ModelPath:  filepath.Join(p.cfg.ModelsDir, opts.Model),
			Language:   opts.Language,
			OutputPath: subtitlePath,
			TempDir:    p.cfg.TempDir,
		}

		p.info("transcribing audio", logging.Field{Key: "input", Value: transReq.InputPath}, logging.Field{Key: "subtitle", Value: subtitlePath})
		transRes, err := p.transcriber.Transcribe(ctx, transReq)
		if err != nil {
			return result, wrapToolErr("transcribe", err)
		}
		subtitlePath = transRes.SubtitlePath
		result.SubtitlePath = subtitlePath
		p.info("transcription complete", logging.Field{Key: "subtitle", Value: subtitlePath})
	}

	if p.renderer == nil {
		return result, ErrNotImplemented
	}

	renderReq := renderer.Request{
		AudioPath:    opts.Input,
		SubtitlePath: subtitlePath,
		OutputPath:   output,
		TempDir:      p.cfg.TempDir,
	}

	p.info("rendering video", logging.Field{Key: "input", Value: renderReq.AudioPath}, logging.Field{Key: "subtitle", Value: renderReq.SubtitlePath}, logging.Field{Key: "output", Value: renderReq.OutputPath})
	renderRes, err := p.renderer.Render(ctx, renderReq)
	if err != nil {
		return result, wrapToolErr("render", err)
	}

	p.info("render complete", logging.Field{Key: "output", Value: renderRes.OutputPath})
	return ConvertResult{OutputPath: renderRes.OutputPath, SubtitlePath: subtitlePath, Steps: steps, DryRun: false}, nil
}

// planSteps determines the sequence of operations based on provided options.
func (p *Pipeline) planSteps(opts ConvertOptions) []string {
	steps := []string{}
	if strings.TrimSpace(opts.Subtitle) == "" {
		steps = append(steps, "transcribe audio")
	} else {
		steps = append(steps, "use provided subtitle")
	}
	steps = append(steps, "render video")
	return steps
}

// BatchOptions bundles parameters for directory-based processing.
type BatchOptions struct {
	InputDir   string
	OutputDir  string
	Extensions []string
	DryRun     bool
	Language   string
	Model      string
}

// BatchJobPlan records how an input file maps to an output artefact.
type BatchJobPlan struct {
	InputPath  string
	OutputPath string
}

// BatchResult summarises the computed workload.
type BatchResult struct {
	Jobs   []BatchJobPlan
	DryRun bool
}

// Batch enumerates the input directory and, in dry-run mode, reports the plan. Non dry-run
// execution iterates jobs sequentially, surfacing ErrNotImplemented while tooling is absent.
func (p *Pipeline) Batch(ctx context.Context, opts BatchOptions) (BatchResult, error) {
	if strings.TrimSpace(opts.InputDir) == "" {
		return BatchResult{}, ErrDirectoryRequired
	}

	outputDir := strings.TrimSpace(opts.OutputDir)
	if outputDir == "" {
		outputDir = p.cfg.OutputDir
	}

	entries, err := os.ReadDir(opts.InputDir)
	if err != nil {
		return BatchResult{}, fmt.Errorf("read input directory: %w", err)
	}

	extensions := opts.Extensions
	if len(extensions) == 0 {
		extensions = defaultExtensions()
	}
	for i, ext := range extensions {
		extensions[i] = strings.ToLower(strings.TrimSpace(ext))
	}

	var jobs []BatchJobPlan
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !containsExtension(extensions, ext) {
			continue
		}

		inputPath := filepath.Join(opts.InputDir, entry.Name())
		base := strings.TrimSuffix(entry.Name(), ext)
		if base == "" {
			base = "output"
		}
		outputPath := filepath.Join(outputDir, base+".mp4")
		jobs = append(jobs, BatchJobPlan{InputPath: inputPath, OutputPath: outputPath})
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].InputPath < jobs[j].InputPath
	})

	result := BatchResult{Jobs: jobs, DryRun: opts.DryRun}
	if opts.DryRun {
		p.debug("batch dry-run", logging.Field{Key: "input", Value: opts.InputDir}, logging.Field{Key: "jobs", Value: len(jobs)})
		return result, nil
	}

	for _, job := range jobs {
		select {
		case <-ctx.Done():
			return BatchResult{}, ctx.Err()
		default:
		}

		_, err := p.Convert(ctx, ConvertOptions{
			Input:    job.InputPath,
			Output:   job.OutputPath,
			Language: opts.Language,
			Model:    opts.Model,
			DryRun:   false,
		})
		if err != nil {
			return result, err
		}
	}

	return BatchResult{Jobs: jobs, DryRun: false}, ErrNotImplemented
}

func (p *Pipeline) debug(msg string, fields ...logging.Field) {
	if p.logger != nil {
		p.logger.Debug(msg, fields...)
	}
}

func (p *Pipeline) info(msg string, fields ...logging.Field) {
	if p.logger != nil {
		p.logger.Info(msg, fields...)
	}
}

func (p *Pipeline) warn(msg string, fields ...logging.Field) {
	if p.logger != nil {
		p.logger.Warn(msg, fields...)
	}
}

func (p *Pipeline) ensureSubtitleProvided(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			p.warn("provided subtitle missing", logging.Field{Key: "subtitle", Value: path})
			return fmt.Errorf("subtitle not found: %w", err)
		}
		return fmt.Errorf("subtitle unavailable: %w", err)
	}
	return nil
}

func defaultExtensions() []string {
	return []string{".mp3", ".wav", ".m4a", ".flac", ".aac", ".ogg"}
}

func containsExtension(list []string, ext string) bool {
	for _, candidate := range list {
		if candidate == ext {
			return true
		}
	}
	return false
}

func wrapToolErr(stage string, err error) error {
	if errors.Is(err, transcriber.ErrNotImplemented) || errors.Is(err, renderer.ErrNotImplemented) {
		return ErrNotImplemented
	}
	return fmt.Errorf("%s stage: %w", stage, err)
}
