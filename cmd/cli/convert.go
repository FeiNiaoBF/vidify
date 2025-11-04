package cli

import (
	"context"
	"errors"
	"time"

	"github.com/spf13/cobra"

	"github.com/FeiNiaoBF/vidify/internal/pipeline"
)

type convertFlags struct {
	input    string
	output   string
	subtitle string
	language string
	model    string
	dryRun   bool
	timeout  time.Duration
}

func (app *application) newConvertCmd() *cobra.Command {
	cfg := &convertFlags{}
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert an audio file into a captioned video",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.runConvert(cmd, cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.input, "input", "i", "", "Path to input audio file")
	cmd.Flags().StringVarP(&cfg.output, "output", "o", "", "Destination video path (defaults to output directory)")
	cmd.Flags().StringVar(&cfg.subtitle, "subtitle", "", "Optional subtitle file to skip transcription")
	cmd.Flags().StringVar(&cfg.language, "language", "", "Language override for transcription")
	cmd.Flags().StringVar(&cfg.model, "model", "", "Whisper model override")
	cmd.Flags().BoolVar(&cfg.dryRun, "dry-run", false, "Plan the conversion without invoking external tools")
	cmd.Flags().DurationVar(&cfg.timeout, "timeout", 0, "Optional timeout for the conversion (e.g. 2m, 45s)")
	_ = cmd.MarkFlagRequired("input")

	return cmd
}

func (app *application) runConvert(cmd *cobra.Command, flags *convertFlags) error {
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

	language := flags.language
	if !cmd.Flags().Changed("language") {
		language = runtime.Config.Runtime.Language
	}

	model := flags.model
	if !cmd.Flags().Changed("model") {
		model = runtime.Config.Runtime.Model
	}

	opts := pipeline.ConvertOptions{
		Input:    flags.input,
		Output:   flags.output,
		Subtitle: flags.subtitle,
		Language: language,
		Model:    model,
		DryRun:   dryRun,
	}

	result, err := runtime.Pipeline.Convert(ctx, opts)
	printConvertResult(cmd, result)
	if err != nil {
		if errors.Is(err, pipeline.ErrNotImplemented) {
			if !result.DryRun && result.SubtitlePath != "" {
				cmd.Println("Note: subtitles generated successfully, but video rendering is not wired yet.")
			} else {
				cmd.Println("Note: external tooling is not wired yet; conversion is planned only.")
			}
			return nil
		}
		return err
	}

	if result.DryRun {
		cmd.Println("Dry-run complete.")
		return nil
	}

	cmd.Printf("Conversion complete: %s\n", result.OutputPath)
	return nil
}

func printConvertResult(cmd *cobra.Command, result pipeline.ConvertResult) {
	if result.OutputPath != "" {
		cmd.Printf("Output: %s\n", result.OutputPath)
	}
	if result.SubtitlePath != "" {
		cmd.Printf("Subtitles: %s\n", result.SubtitlePath)
	}
	if len(result.Steps) == 0 {
		return
	}
	cmd.Println("Planned steps:")
	for _, step := range result.Steps {
		cmd.Printf("  - %s\n", step)
	}
}
