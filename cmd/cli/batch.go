package cli

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/FeiNiaoBF/vidify/internal/pipeline"
)

type batchFlags struct {
	inputDir   string
	outputDir  string
	extensions []string
	dryRun     bool
	timeout    time.Duration
}

func (app *application) newBatchCmd() *cobra.Command {
	cfg := &batchFlags{}
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Batch convert audio files within a directory",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.runBatch(cmd, cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.inputDir, "input-dir", "d", "", "Directory containing audio files")
	cmd.Flags().StringVarP(&cfg.outputDir, "output-dir", "o", "", "Directory for generated videos")
	cmd.Flags().StringSliceVar(&cfg.extensions, "include-ext", nil, "Audio file extensions to include (e.g. .mp3,.wav)")
	cmd.Flags().BoolVar(&cfg.dryRun, "dry-run", false, "Plan the workload without invoking external tools")
	cmd.Flags().DurationVar(&cfg.timeout, "timeout", 0, "Optional timeout for the batch job")
	_ = cmd.MarkFlagRequired("input-dir")

	return cmd
}

func (app *application) runBatch(cmd *cobra.Command, flags *batchFlags) error {
	runtime, ok := runtimeFromContext(cmd.Context())
	if !ok {
		return errors.New("runtime not initialised")
	}

	ctx := cmd.Context()
	if flags.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, flags.timeout)
		defer cancel()
	}

	dryRun := flags.dryRun
	if !cmd.Flags().Changed("dry-run") {
		dryRun = runtime.Config.Runtime.DryRun
	}

	language := runtime.Config.Runtime.Language
	model := runtime.Config.Runtime.Model

	opts := pipeline.BatchOptions{
		InputDir:   flags.inputDir,
		OutputDir:  flags.outputDir,
		Extensions: flags.extensions,
		DryRun:     dryRun,
		Language:   language,
		Model:      model,
	}

	result, err := runtime.Pipeline.Batch(ctx, opts)
	printBatchResult(cmd, result)
	if err != nil {
		if errors.Is(err, pipeline.ErrNotImplemented) {
			cmd.Println("Note: external tooling is not wired yet; batch run is planned only.")
			return nil
		}
		return err
	}

	if result.DryRun {
		cmd.Println("Dry-run complete.")
		return nil
	}

	cmd.Printf("Batch conversion complete for %d file(s).\n", len(result.Jobs))
	return nil
}

func printBatchResult(cmd *cobra.Command, result pipeline.BatchResult) {
	if len(result.Jobs) == 0 {
		cmd.Println("No candidate audio files found. Adjust filters or input directory.")
		return
	}
	cmd.Println("Planned batch jobs:")
	for _, job := range result.Jobs {
		cmd.Printf("  - %s -> %s\n", job.InputPath, job.OutputPath)
	}
}
