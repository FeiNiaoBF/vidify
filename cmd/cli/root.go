package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "vidify",
	Short: "vidify is a tool to create videos from audio",
	Long: `vidify is a tool to create videos from audio. Use whisper to generate subtitles from audio, and then use ffmpeg to create videos.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Execute failed:", err)
		os.Exit(1)
	}
}
