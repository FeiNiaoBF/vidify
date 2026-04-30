package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// huggingFaceBase is the canonical Whisper model repository on HuggingFace.
const huggingFaceBase = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"

// modelCmd is the parent `vidify model` command.
var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Manage local Whisper model files",
}

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List downloaded Whisper models",
	RunE:  runModelList,
}

var modelDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a Whisper model from HuggingFace",
	RunE:  runModelDownload,
}

func init() {
	rootCmd.AddCommand(modelCmd)
	modelCmd.AddCommand(modelListCmd)
	modelCmd.AddCommand(modelDownloadCmd)

	modelDownloadCmd.Flags().StringP("model", "m", "base", "model size: tiny|base|small|medium|large")
	modelDownloadCmd.Flags().StringP("output-dir", "o", "", "destination directory (default: ~/.vidify/models)")
}

// runModelList scans the models directory and prints each .bin file it finds.
// It does not import appdir — the path is constructed inline to respect the
// architectural constraint that adapter/cli must not import infrastructure packages.
func runModelList(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	modelsDir := filepath.Join(home, ".vidify", "models")

	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No models directory found. Run `vidify init` first.")
			return nil
		}
		return fmt.Errorf("read models directory: %w", err)
	}

	var found int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".bin") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		fmt.Printf("  %-30s  %s\n", e.Name(), formatBytes(info.Size()))
		found++
	}

	if found == 0 {
		fmt.Println("No models found. Run `vidify model download --model base` to get started.")
	} else {
		fmt.Printf("\n%d model(s) in %s\n", found, modelsDir)
	}
	return nil
}

// runModelDownload downloads the requested model from HuggingFace with an
// mpb progress bar. Ctrl+C cancels the download cleanly via context.
func runModelDownload(cmd *cobra.Command, _ []string) error {
	modelName, _ := cmd.Flags().GetString("model")
	outputDir, _ := cmd.Flags().GetString("output-dir")

	if outputDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		outputDir = filepath.Join(home, ".vidify", "models")
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create models directory: %w", err)
	}

	filename := fmt.Sprintf("ggml-%s.bin", modelName)
	destPath := filepath.Join(outputDir, filename)
	url := fmt.Sprintf("%s/%s", huggingFaceBase, filename)

	// Use a context tied to SIGINT/SIGTERM (already set up by the parent
	// command's cobra.OnInitialize chain). Here we rely on cmd.Context() which
	// cobra populates from the root context when ExecuteContext is used, but
	// fall back to Background if not set.
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	fmt.Printf("Downloading %s\n  from: %s\n  to:   %s\n\n", modelName, url, destPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s for %s", resp.Status, url)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create destination file: %w", err)
	}
	defer out.Close()

	// Show download progress with mpb.
	p := mpb.New(mpb.WithWidth(60))
	bar := p.AddBar(resp.ContentLength,
		mpb.PrependDecorators(
			decor.Name(filename, decor.WCSyncSpaceR),
			decor.CountersKibiByte("%.1f / %.1f"),
		),
		mpb.AppendDecorators(
			decor.EwmaETA(decor.ET_STYLE_GO, 30),
			decor.Name(" ] "),
			decor.EwmaSpeed(decor.SizeB1024(0), "% .2f", 30),
		),
	)

	reader := bar.ProxyReader(resp.Body)
	defer reader.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return fmt.Errorf("write model file: %w", err)
	}

	bar.SetTotal(-1, true)
	p.Wait()

	fmt.Printf("\nSaved to %s\n", destPath)
	return nil
}

// formatBytes converts a byte count into a human-readable string (KB / MB / GB).
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
