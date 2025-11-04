package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FeiNiaoBF/vidify/internal/buildinfo"
	"github.com/FeiNiaoBF/vidify/internal/config"
	"github.com/FeiNiaoBF/vidify/internal/logging"
	"github.com/FeiNiaoBF/vidify/internal/pipeline"
	"github.com/FeiNiaoBF/vidify/internal/tools/renderer"
	"github.com/FeiNiaoBF/vidify/internal/tools/transcriber"
)

// Dependencies groups required collaborators to build the CLI application.
type Dependencies struct {
	Loader      *config.Loader
	LogFactory  logging.Factory
	Transcriber transcriber.Service
	Renderer    renderer.Service
}

// New constructs the root Cobra command with all subcommands wired.
func New(deps Dependencies) (*cobra.Command, error) {
	if deps.Loader == nil {
		return nil, errors.New("config loader is required")
	}

	app := &application{deps: deps}
	root := &cobra.Command{
		Use:          "vidify",
		Short:        "Convert audio into captioned videos",
		SilenceUsage: true,
		Version:      buildinfo.Version,
		// before any command runs, ensure runtime is initialized
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.ensureRuntime(cmd)
		},
		// after any command runs
		PersistentPostRun: func(cmd *cobra.Command, _ []string) {
			app.teardown(cmd)
		},
	}
	root.SetVersionTemplate("vidify {{.Version}}\n")

	root.PersistentFlags().String("config", "", "Path to configuration file")
	root.PersistentFlags().String("log-level", "", "Log level override (debug, info, warn, error)")
	root.PersistentFlags().String("log-mode", "", "Log mode (development or production)")

	_ = deps.Loader.BindFlag(root.PersistentFlags().Lookup("log-level"), "logging.level")
	_ = deps.Loader.BindFlag(root.PersistentFlags().Lookup("log-mode"), "logging.mode")

	convertCmd := app.newConvertCmd()
	batchCmd := app.newBatchCmd()
	versionCmd := app.newVersionCmd()

	root.AddCommand(convertCmd, batchCmd, versionCmd)

	return root, nil
}

// application holds runtime state helpers.
type application struct {
	deps Dependencies
}

type runtimeKey struct{}

type runtimeState struct {
	Config   config.AppConfig
	Logger   logging.Logger
	Pipeline *pipeline.Pipeline
}

// ensureRuntime initializes runtime state if not already present in the command context.
// the method loads configuration, sets up logging, and prepares the processing pipeline.
func (app *application) ensureRuntime(cmd *cobra.Command) error {
	if _, ok := runtimeFromContext(cmd.Context()); ok {
		return nil
	}

	cfgPath, _ := cmd.Flags().GetString("config")
	cfg, err := app.deps.Loader.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if flag := cmd.PersistentFlags().Lookup("log-level"); flag != nil && flag.Changed {
		s := flag.Value.String()
		if s != "" {
			cfg.Logging.Level = s
		}
	}
	if flag := cmd.PersistentFlags().Lookup("log-mode"); flag != nil && flag.Changed {
		s := flag.Value.String()
		if s != "" {
			cfg.Logging.Mode = s
		}
	}

	logger, err := app.deps.LogFactory.Build(logging.Config{
		Level: cfg.Logging.Level,
		Mode:  cfg.Logging.Mode,
	})
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	// initialize pipeline dependencies
	transSvc := app.deps.Transcriber
	if transSvc == nil {
		transSvc, err = transcriber.NewCLI(transcriber.CLIOptions{
			BinaryPath: cfg.Tools.Whisper,
			FFmpegPath: cfg.Tools.FFmpeg,
			Env:        cfg.Runtime.Env,
		})
		if err != nil {
			return fmt.Errorf("init transcriber: %w", err)
		}
	}

	pipe := pipeline.New(pipeline.Config{
		OutputDir: cfg.Paths.Output,
		TempDir:   cfg.Paths.Temp,
		ModelsDir: cfg.Paths.Models,
	}, pipeline.Dependencies{
		Transcriber: transSvc,
		Renderer:    app.deps.Renderer,
		Logger:      logger,
	})

	runtime := runtimeState{Config: cfg, Logger: logger, Pipeline: pipe}
	ctx := context.WithValue(cmd.Context(), runtimeKey{}, runtime)
	cmd.SetContext(ctx)

	return nil
}

func (app *application) teardown(cmd *cobra.Command) {
	runtime, ok := runtimeFromContext(cmd.Context())
	if !ok {
		return
	}
	if runtime.Logger != nil {
		_ = runtime.Logger.Sync()
	}
}

func runtimeFromContext(ctx context.Context) (runtimeState, bool) {
	val := ctx.Value(runtimeKey{})
	if val == nil {
		return runtimeState{}, false
	}
	rt, ok := val.(runtimeState)
	return rt, ok
}
