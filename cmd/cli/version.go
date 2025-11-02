package cli

import (
	"github.com/spf13/cobra"

	"github.com/FeiNiaoBF/vidify/internal/buildinfo"
)

func (app *application) newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print vidify build information",
		Run: func(cmd *cobra.Command, _ []string) {
			runtime, _ := runtimeFromContext(cmd.Context())
			_ = runtime // runtime placeholder for future extended output
			cmd.Println(buildinfo.Summary())
		},
	}
}
