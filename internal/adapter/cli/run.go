package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
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
	c, err := di.Build(viper.GetViper(), resolveWhisperBin(), resolveFFmpegBin())
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Flush the logger's lumberjack file on exit so no log lines are lost.
	if c.Logger != nil {
		defer c.Logger.Close()
	}

	// Cancel the context on SIGINT/SIGTERM so child processes (whisper-cli,
	// ffmpeg) are killed promptly when the user presses Ctrl+C.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var results []port.Result

	// --file-list / -F: read one input path per line and process each.
	if fileList := viper.GetString("file-list"); fileList != "" {
		paths, err := readPathsFromFile(fileList)
		if err != nil {
			return fmt.Errorf("read file-list %q: %w", fileList, err)
		}
		for _, p := range paths {
			cfg := c.Config
			cfg.InputPath = p
			results = append(results, c.Processor.ProcessFile(ctx, cfg))
		}
	} else {
		info, err := os.Stat(c.Config.InputPath)
		if err != nil {
			return fmt.Errorf("cannot access input path %q: %w", c.Config.InputPath, err)
		}

		if info.IsDir() {
			results = c.Processor.ProcessBatch(ctx, c.Config)
		} else {
			results = []port.Result{c.Processor.ProcessFile(ctx, c.Config)}
		}
	}

	// Block until all progress bar rendering is flushed before printing the
	// summary — otherwise the summary line can interleave with bar output.
	// We type-assert via any() rather than hard-wiring the *progress.Reporter
	// type so that a future GUI implementation without Wait() compiles cleanly.
	if w, ok := any(c.Progress).(interface{ Wait() }); ok {
		w.Wait()
	}

	return printResults(results)
}

// readPathsFromFile reads a newline-delimited list of file paths from path.
// Blank lines and lines starting with '#' are ignored.
func readPathsFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var paths []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		paths = append(paths, line)
	}
	return paths, sc.Err()
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
