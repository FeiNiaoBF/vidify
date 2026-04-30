package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// doctorCmd is the `vidify doctor` command.
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the environment for required tools and directories",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// runDoctor performs a series of environment checks and prints a ✓/✗ report.
// It is intentionally read-only and makes no changes to the system.
func runDoctor(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	vidifyDir := filepath.Join(home, ".vidify")

	fmt.Println("Vidify environment check")
	fmt.Println(strings.Repeat("─", 40))

	checkBinary("whisper-cli", resolveWhisperBin(), "--version",
		"ensure runtime-tools/whisper/<platform>/whisper-cli is next to the vidify binary\n"+
			"      (from your release archive), or set whisper-bin in ~/.vidify/config.yaml")
	checkBinary("ffmpeg", resolveFFmpegBin(), "-version",
		"ensure runtime-tools/ffmpeg/<platform>/ffmpeg is next to the vidify binary\n"+
			"      (from your release archive), or set ffmpeg-bin in ~/.vidify/config.yaml")
	checkDir("~/.vidify/", vidifyDir)
	checkModels(filepath.Join(vidifyDir, "models"))
	checkDir("~/.vidify/logs/", filepath.Join(vidifyDir, "logs"))

	return nil
}

// checkBinary verifies that the binary exists and, if so, runs a version flag
// to confirm the binary is executable. The first line of output is printed.
// fixHint is shown to the user when the binary cannot be found.
func checkBinary(label, bin, versionFlag, fixHint string) {
	if _, err := os.Stat(bin); os.IsNotExist(err) {
		// Stat failed — the path is bare (not bundled). Try PATH lookup.
		if _, err := exec.LookPath(bin); err != nil {
			printCheck(false, label, "not found — "+fixHint)
			return
		}
	}

	out, err := exec.Command(bin, versionFlag).CombinedOutput()
	if err != nil {
		printCheck(false, label, fmt.Sprintf("found at %s but failed to run: %v", bin, err))
		return
	}

	// Print the first non-empty line as the version string.
	firstLine := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	printCheck(true, label, fmt.Sprintf("%s  (%s)", bin, firstLine))
}

// checkDir verifies that a directory exists on disk.
func checkDir(label, path string) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		printCheck(true, label, path)
	} else {
		printCheck(false, label, fmt.Sprintf("%s — run `vidify init` to create it", path))
	}
}

// checkModels counts .bin files in the models directory.
func checkModels(modelsDir string) {
	entries, err := os.ReadDir(modelsDir)
	if err != nil {
		printCheck(false, "models/", fmt.Sprintf("%s — run `vidify model download --model base`", modelsDir))
		return
	}

	var count int
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".bin") {
			count++
		}
	}

	if count == 0 {
		printCheck(false, "models/", fmt.Sprintf("0 models in %s — run `vidify model download --model base`", modelsDir))
	} else {
		printCheck(true, "models/", fmt.Sprintf("%d model(s) in %s", count, modelsDir))
	}
}

// printCheck prints a single ✓ or ✗ check line to stdout.
func printCheck(ok bool, label, detail string) {
	mark := "✓"
	if !ok {
		mark = "✗"
	}
	fmt.Printf("  %s  %-20s %s\n", mark, label, detail)
}
