// Package logger provides the infrastructure adapter for structured logging.
//
// It wraps log/slog behind the port.Logger interface so that all application
// and use-case code depends only on the interface, not on the slog package.
// This makes it possible to swap backends (zerolog, zap, etc.) without
// touching any code outside this package.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/pkg/errors"

	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// Config controls the logger's runtime behaviour.
// Zero value produces an Info-level, text-format, stderr logger.
type Config struct {
	// Level is the minimum log severity: "debug", "info", "warn", "error".
	// Empty string defaults to "info".
	Level string

	// Format selects the output format: "text" (default) or "json".
	// JSON output is preferable for log aggregation pipelines.
	Format string

	// File is an optional filesystem path for log output.
	// Empty string means write to the writer passed to New/NewWithConfig.
	// A misconfigured path returns an error rather than silently falling
	// back to stderr, because a lost log stream is a production bug.
	File string
}

// slogLogger wraps *slog.Logger to implement port.Logger.
// The struct is unexported — callers only interact via the port.Logger interface.
type slogLogger struct {
	inner *slog.Logger
}

// New creates a port.Logger writing to w.
// When verbose is true the level is set to Debug; otherwise Info.
// For fine-grained control (JSON format, file output) use NewWithConfig.
func New(w io.Writer, verbose bool) port.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	return build(w, level, false)
}

// NewWithConfig creates a port.Logger from an explicit Config.
// The caller owns closing any file handle opened for cfg.File;
// currently Vidify runs as a short-lived CLI so we intentionally
// leave the file open until process exit.
func NewWithConfig(cfg Config) (port.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, errors.Wrap(err, "invalid log level")
	}

	w := io.Writer(os.Stderr)
	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot open log file %q", cfg.File)
		}
		w = f
	}

	return build(w, level, cfg.Format == "json"), nil
}

func build(w io.Writer, level slog.Level, useJSON bool) port.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler
	if useJSON {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return &slogLogger{inner: slog.New(h)}
}

// The four log-level methods satisfy port.Logger.
// args follows slog conventions: alternating key/value pairs or slog.Attr values.
func (l *slogLogger) Debug(msg string, args ...any) { l.inner.Debug(msg, args...) }
func (l *slogLogger) Info(msg string, args ...any)  { l.inner.Info(msg, args...) }
func (l *slogLogger) Warn(msg string, args ...any)  { l.inner.Warn(msg, args...) }
func (l *slogLogger) Error(msg string, args ...any) { l.inner.Error(msg, args...) }

// With returns a new Logger with the given key-value pairs added to every
// subsequent entry. We return port.Logger (not *slogLogger) so callers
// stay decoupled from the concrete type.
func (l *slogLogger) With(args ...any) port.Logger {
	return &slogLogger{inner: l.inner.With(args...)}
}

func parseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q (want: debug|info|warn|error)", s)
	}
}
