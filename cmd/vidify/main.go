package main

import (
	"context"
	"fmt"
	"os"

	"github.com/FeiNiaoBF/vidify/cmd/cli"
	"github.com/FeiNiaoBF/vidify/internal/config"
	"github.com/FeiNiaoBF/vidify/internal/logging"
	"github.com/FeiNiaoBF/vidify/internal/tools/renderer"
)

func main() {
	loader := config.NewLoader("VIDIFY")
	factory := logging.NewFactory()

	rootCmd, err := cli.New(cli.Dependencies{
		Loader:     loader,
		LogFactory: factory,
		Renderer:   renderer.NewStub(),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "init failed:", err)
		os.Exit(1)
	}

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "execute failed:", err)
		os.Exit(1)
	}
}
