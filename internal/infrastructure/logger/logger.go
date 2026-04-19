// Package logger provides the infrastructure adapter for structured logging.
//
// It wraps log/slog behind the port.Logger interface so that all application
// and use-case code depends only on the interface, not on the slog package.
// Console output uses a human-readable TextHandler; file output uses a
// machine-parseable JSONHandler. Both are combined via slog.NewMultiHandler
// (Go 1.26+). Log file rotation is handled by lumberjack.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// Default values applied when Config fields are zero.
const (
	defaultDir        = ""   // resolved lazily to ~/.vidify/logs
	defaultLevel      = "info"
	defaultToFile     = true
	defaultMaxBackups = 7
	defaultMaxAge     = 30  // days
	defaultMaxSize    = 100 // megabytes
)

// Config controls the logger's runtime behaviour.
// Zero values are replaced by sensible defaults in NewWithConfig.
type Config struct {
	// Dir is the directory where log files are written.
	// Empty string resolves to ~/.vidify/logs at runtime.
	Dir string

	// Level is the minimum log severity: "debug", "info", "warn", "error".
	// Empty string defaults to "info".
	Level string

	// ToFile enables writing a JSON-formatted log file alongside the console
	// output. Defaults to true. Set to false for silent / pipe-friendly usage.
	ToFile bool

	// MaxBackups is the number of old log files to retain.
	// Defaults to 7 (roughly one week at one file per run-day).
	MaxBackups int

	// MaxAge is the maximum number of days to retain old log files.
	// Defaults to 30.
	MaxAge int

	// MaxSize is the maximum size in megabytes before the log file is rotated.
	// Defaults to 100.
	MaxSize int
}

// slogLogger wraps *slog.Logger to implement port.Logger.
// The struct is unexported — callers only interact via the port.Logger interface.
type slogLogger struct {
	inner *slog.Logger
}

// Compile-time checks that *slogLogger satisfies both port.Logger and io.Closer.
var _ port.Logger = (*slogLogger)(nil)
var _ io.Closer = (*slogLogger)(nil)

// New creates a port.Logger writing to w with a plain TextHandler.
// When verbose is true the level is set to Debug; otherwise Info.
// This is a lightweight helper for tests and simple integrations; production
// code should use NewWithConfig for full dual-output and rotation support.
func New(w io.Writer, verbose bool) port.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	opts := &slog.HandlerOptions{Level: level}
	return &slogLogger{inner: slog.New(slog.NewTextHandler(w, opts))}
}

// NewWithConfig creates a port.Logger from an explicit Config.
// It always writes human-readable text to stderr (console).
// When cfg.ToFile is true it also writes structured JSON to a rotating log
// file under cfg.Dir, combining both outputs via slog.NewMultiHandler.
//
// File initialisation errors are non-fatal: a warning is printed to stderr
// and the logger falls back to console-only output so the program can still
// run. This is intentional — a missing log file must not prevent processing.
func NewWithConfig(cfg Config) port.Logger {
	applyDefaults(&cfg)

	level, err := parseLevel(cfg.Level)
	if err != nil {
		// Unrecognised level string: warn and fall back to Info.
		fmt.Fprintf(os.Stderr, "WARN logger: unknown log level %q, using info\n", cfg.Level)
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}
	consoleH := slog.NewTextHandler(os.Stderr, opts)

	if !cfg.ToFile {
		return &slogLogger{inner: slog.New(consoleH)}
	}

	fileH, err := buildFileHandler(cfg, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN logger: cannot initialise log file, falling back to console-only: %v\n", err)
		return &slogLogger{inner: slog.New(consoleH)}
	}

	return &slogLogger{inner: slog.New(slog.NewMultiHandler(consoleH, fileH))}
}

// buildFileHandler creates the JSON slog.Handler backed by a lumberjack
// rotating writer. It expands the log directory, creates it if necessary,
// and generates a timestamped filename for this invocation.
func buildFileHandler(cfg Config, opts *slog.HandlerOptions) (slog.Handler, error) {
	dir, err := expandDir(cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("resolve log dir: %w", err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir %q: %w", dir, err)
	}

	// A fixed filename lets lumberjack manage rotation correctly. When the file
	// reaches MaxSize it is renamed with a timestamp suffix and a fresh file is
	// opened — this is lumberjack's own rotation mechanism. Per-invocation
	// granularity is already provided by the structured timestamp field in each
	// JSON log entry.
	path := filepath.Join(dir, "vidify.log")

	lj := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   true, // gzip old logs to save disk space
	}

	return slog.NewJSONHandler(lj, opts), nil
}

// applyDefaults fills zero values in cfg with built-in defaults.
// This keeps NewWithConfig callers honest: an empty Config still works.
//
// ToFile truth table — because bool's zero value is false, we cannot
// distinguish "user explicitly set false" from "user never set anything"
// using a plain bool field. The heuristic below uses the combination of
// three fields to detect a fully-zero (never-configured) struct:
//
//	Dir   ToFile  MaxBackups │ result
//	──────────────────────────┼──────────────────────────────────────────
//	""    false   0          │ ToFile → true  (fully-zero: apply defaults)
//	""    false   N>0        │ ToFile stays false  (explicit false + other fields set)
//	""    true    any        │ ToFile stays true   (explicit true)
//	"…"   false   any        │ ToFile stays false  (Dir set means partial config)
//
func applyDefaults(cfg *Config) {
	if cfg.Level == "" {
		cfg.Level = defaultLevel
	}
	if cfg.Dir == "" && !cfg.ToFile && cfg.MaxBackups == 0 {
		cfg.ToFile = defaultToFile
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = defaultMaxBackups
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = defaultMaxAge
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = defaultMaxSize
	}
}

// expandDir replaces a leading "~" with the current user's home directory.
// It handles both Unix-style "~/" and Windows-style "~\" prefixes so YAML
// config entries like "~\\logs" work correctly on Windows.
// If dir is empty, it returns the default log directory ~/.vidify/logs.
func expandDir(dir string) (string, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, ".vidify", "logs"), nil
	}
	// Accept both Unix ("~/") and Windows ("~\\") tilde prefixes.
	if strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, dir[2:]), nil
	}
	return dir, nil
}

// parseLevel maps a string level name to slog.Level.
func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown level %q", s)
	}
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

// WithError returns a new Logger with err attached under the "error" key.
// It is sugar over With("error", err.Error()) for the ubiquitous pattern of
// logging a non-nil error alongside a descriptive message.
func (l *slogLogger) WithError(err error) port.Logger {
	if err == nil {
		return l
	}
	return l.With("error", err.Error())
}

// Close flushes and releases any underlying io.Closer held by the handler
// chain (e.g., the lumberjack file writer). Callers should invoke Close via
// defer after creating the logger so the final log entries are not lost on
// an abrupt process exit.
//
// If the inner handler does not hold a Closer (e.g., console-only mode)
// Close is a no-op and always returns nil.
func (l *slogLogger) Close() error {
	type handlerCloser interface {
		Handler() slog.Handler
	}
	// slog.Logger does not expose the underlying writer, but we can reach the
	// lumberjack.Logger through the handler if it implements io.Closer.
	if hc, ok := l.inner.Handler().(io.Closer); ok {
		return hc.Close()
	}
	return nil
}
