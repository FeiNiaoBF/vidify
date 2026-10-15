package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

func newProcessCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "process",
		Short: "Generate a subtitle video and an SRT file from audio",
		Long: `Generate a subtitle video and an SRT file from one audio file.

Audio processing and its command-line options are not implemented yet.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.New("audio processing is not implemented yet")
		},
	}
}
