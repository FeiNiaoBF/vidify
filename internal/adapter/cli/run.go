package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/FeiNiaoBF/vidify/internal/di"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// runRoot is the CLI command handler. It delegates all wiring to di.Build so
// that this file only contains presentation-layer concerns: flag resolution,
// signal handling, and result formatting.
func runRoot(cmd *cobra.Command, args []string) error {
	c, err := di.Build(viper.GetViper(), resolveWhisperBin())
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Cancel the context on SIGINT/SIGTERM so child processes (whisper-cli,
	// ffmpeg) are killed promptly when the user presses Ctrl+C.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	info, err := os.Stat(c.Config.InputPath)
	if err != nil {
		return fmt.Errorf("cannot access input path %q: %w", c.Config.InputPath, err)
	}

	var results []port.Result
	if info.IsDir() {
		results = c.Processor.ProcessBatch(ctx, c.Config)
	} else {
		results = []port.Result{c.Processor.ProcessFile(ctx, c.Config)}
	}

	// Block until all progress bar rendering is flushed before printing the
	// summary — otherwise the summary line can interleave with bar output.
	c.Progress.Wait()

	return printResults(results)
}

// printResults outputs a human-readable summary of all processing results.
func printResults(results []port.Result) error {
	var failed int
	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "FAIL  %s — %v\n", r.InputPath, r.Err)
			failed++
		} else {
			fmt.Fprintf(os.Stdout, "OK    %s → %s\n", r.InputPath, r.OutputPath)
		}
	}

	fmt.Fprintf(os.Stdout, "\nTotal: %d  Success: %d  Failed: %d\n",
		len(results), len(results)-failed, failed)

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed to process", failed)
	}
	return nil
}
