// Package main is the entry point for the vidify CLI.
// It stays minimal: wire dependencies, then delegate to cobra.
package main

import (
	"os"

	"github.com/FeiNiaoBF/vidify/internal/adapter/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
