package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "vidify",
		Short:         "Create subtitle videos from audio",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newProcessCommand())
	return cmd
}

func Execute() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
