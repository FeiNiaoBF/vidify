package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := newRootCommand().ExecuteContext(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
