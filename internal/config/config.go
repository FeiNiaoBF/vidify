package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// AppConfig groups the configuration sections consumed across the application layers.
type AppConfig struct {
	Paths   PathsConfig   `mapstructure:"paths"`
	Tools   ToolsConfig   `mapstructure:"tools"`
	Runtime RuntimeConfig `mapstructure:"runtime"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// PathsConfig captures filesystem layout for inputs and outputs.
type PathsConfig struct {
	Temp   string `mapstructure:"temp"`
	Output string `mapstructure:"output"`
	Models string `mapstructure:"models"` // for whisper models
}

// ToolsConfig stores external binary locations.
type ToolsConfig struct {
	FFmpeg  string `mapstructure:"ffmpeg"`
	Whisper string `mapstructure:"whisper"`
}

// RuntimeConfig defines operational defaults for CLI workflows.
type RuntimeConfig struct {
	Language string            `mapstructure:"language"`
	Model    string            `mapstructure:"model"`
	DryRun   bool              `mapstructure:"dry_run"`
	Env      map[string]string `mapstructure:"env"`
}

// LoggingConfig governs logger initialisation.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	Mode  string `mapstructure:"mode"`
}

// Loader encapsulates viper usage to enforce precedence: flags > env > config > defaults.
type Loader struct {
	v         *viper.Viper
	envPrefix string
}

// NewLoader constructs a loader with sane defaults.
func NewLoader(envPrefix string) *Loader {
	v := viper.New()
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	loader := &Loader{v: v, envPrefix: envPrefix}
	loader.setDefaults()
	return loader
}

func (l *Loader) setDefaults() {
	l.v.SetDefault("paths.temp", "temp")
	l.v.SetDefault("paths.output", "output")
	l.v.SetDefault("paths.models", "models")
	l.v.SetDefault("tools.ffmpeg", "ffmpeg")
	l.v.SetDefault("tools.whisper", "whisper-cli")
	l.v.SetDefault("runtime.language", "auto")
	l.v.SetDefault("runtime.model", "ggml-base.en.bin")
	l.v.SetDefault("runtime.dry_run", true)
	l.v.SetDefault("runtime.env", map[string]string{})
	l.v.SetDefault("logging.level", "info")
	l.v.SetDefault("logging.mode", "development")
}

// BindFlags links the provided flag set to the loader using the supplied key prefix.
func (l *Loader) BindFlags(fs *pflag.FlagSet, keyPrefix string) error {
	if fs == nil {
		return nil
	}
	var err error
	fs.VisitAll(func(f *pflag.Flag) {
		if err != nil {
			return
		}
		key := f.Name
		if keyPrefix != "" {
			key = keyPrefix + "." + key
		}
		err = l.v.BindPFlag(key, f)
	})
	return err
}

// BindFlag binds a single flag to the supplied configuration key.
func (l *Loader) BindFlag(flag *pflag.Flag, key string) error {
	if flag == nil {
		return nil
	}
	return l.v.BindPFlag(key, flag)
}

// Load reads configuration from the supplied path (if any) and produces AppConfig.
func (l *Loader) Load(configPath string) (AppConfig, error) {
	if strings.TrimSpace(configPath) != "" {
		l.v.SetConfigFile(configPath)
		if err := l.v.ReadInConfig(); err != nil {
			return AppConfig{}, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg AppConfig
	if err := l.v.Unmarshal(&cfg); err != nil {
		return AppConfig{}, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}

// EnvPrefix exposes the configured environment-variable prefix.
func (l *Loader) EnvPrefix() string {
	return l.envPrefix
}
