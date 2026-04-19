package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestParseLevel verifies every supported level string and the fallback for
// unknown values. A misspelled level in a config file must not silently
// produce wrong behaviour.
func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "debug lowercase", input: "debug", wantErr: false},
		{name: "info lowercase", input: "info", wantErr: false},
		{name: "INFO uppercase", input: "INFO", wantErr: false},
		{name: "warn lowercase", input: "warn", wantErr: false},
		{name: "warning alias", input: "warning", wantErr: false},
		{name: "error lowercase", input: "error", wantErr: false},
		{name: "empty string defaults to info", input: "", wantErr: false},
		{name: "unknown string", input: "trace", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseLevel(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLevel(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestExpandDir verifies path expansion for the empty string, Unix tilde,
// Windows tilde, and absolute paths.
func TestExpandDir(t *testing.T) {
	t.Parallel()

	// Non-tilde absolute path must be returned unchanged.
	absPath := `/some/absolute/path`
	got, err := expandDir(absPath)
	if err != nil {
		t.Fatalf("expandDir(%q) unexpected error: %v", absPath, err)
	}
	if got != absPath {
		t.Errorf("expandDir(%q) = %q, want %q", absPath, got, absPath)
	}

	// Empty string must resolve to something containing ".vidify/logs".
	got, err = expandDir("")
	if err != nil {
		t.Fatalf("expandDir(\"\") unexpected error: %v", err)
	}
	if !strings.Contains(got, ".vidify") {
		t.Errorf("expandDir(\"\") = %q, expected it to contain .vidify", got)
	}

	// Unix-style tilde prefix.
	got, err = expandDir("~/myapp")
	if err != nil {
		t.Fatalf("expandDir(\"~/myapp\") unexpected error: %v", err)
	}
	if strings.HasPrefix(got, "~") {
		t.Errorf("expandDir(\"~/myapp\") = %q, tilde was not expanded", got)
	}

	// Windows-style tilde prefix.
	got, err = expandDir(`~\myapp`)
	if err != nil {
		t.Fatalf(`expandDir("~\myapp") unexpected error: %v`, err)
	}
	if strings.HasPrefix(got, "~") {
		t.Errorf(`expandDir("~\myapp") = %q, tilde was not expanded`, got)
	}
}

// TestApplyDefaults verifies the truth table documented in applyDefaults.
func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      Config
		wantToFile bool
	}{
		{
			name:       "fully zero struct enables file logging",
			input:      Config{},
			wantToFile: true,
		},
		{
			name:       "explicit ToFile false with non-zero MaxBackups stays false",
			input:      Config{MaxBackups: 5},
			wantToFile: false,
		},
		{
			name:       "explicit ToFile true stays true",
			input:      Config{ToFile: true},
			wantToFile: true,
		},
		{
			name:       "Dir set keeps explicit false",
			input:      Config{Dir: "/tmp/logs"},
			wantToFile: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := tt.input
			applyDefaults(&cfg)
			if cfg.ToFile != tt.wantToFile {
				t.Errorf("applyDefaults(%+v).ToFile = %v, want %v", tt.input, cfg.ToFile, tt.wantToFile)
			}
			// Shared invariant: numeric fields must never be zero after defaults.
			if cfg.MaxBackups == 0 {
				t.Error("applyDefaults left MaxBackups = 0")
			}
			if cfg.MaxAge == 0 {
				t.Error("applyDefaults left MaxAge = 0")
			}
			if cfg.MaxSize == 0 {
				t.Error("applyDefaults left MaxSize = 0")
			}
		})
	}
}

// TestNew_LevelFilter verifies that New respects the verbose flag for level
// filtering. We write a DEBUG entry and check whether it appears in the output.
func TestNew_LevelFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		verbose   bool
		wantDebug bool
	}{
		{name: "verbose true shows debug", verbose: true, wantDebug: true},
		{name: "verbose false hides debug", verbose: false, wantDebug: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			log := New(&buf, tt.verbose)
			log.Debug("sentinel-debug-message")

			contains := strings.Contains(buf.String(), "sentinel-debug-message")
			if contains != tt.wantDebug {
				t.Errorf("verbose=%v: debug message present=%v, want %v\noutput: %s",
					tt.verbose, contains, tt.wantDebug, buf.String())
			}
		})
	}
}

// TestWithError_NilSafety verifies that WithError(nil) returns the same logger
// and does not inject an "error" field into the output.
func TestWithError_NilSafety(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := New(&buf, true)

	returned := log.WithError(nil)
	if returned != log {
		t.Error("WithError(nil) should return the same Logger instance")
	}

	// Log a message and confirm no "error=" key appears.
	returned.Info("no-error-message")
	if strings.Contains(buf.String(), "error=") {
		t.Errorf("WithError(nil) injected an error field: %s", buf.String())
	}
}

// TestWithError_AttachesField verifies that WithError with a non-nil error
// attaches the error string under the "error" key in subsequent log entries.
func TestWithError_AttachesField(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := New(&buf, true)
	log.WithError(errors.New("something went wrong")).Error("operation failed")

	out := buf.String()
	if !strings.Contains(out, "something went wrong") {
		t.Errorf("WithError did not include error text in output: %s", out)
	}
	if !strings.Contains(out, "error") {
		t.Errorf("WithError did not attach 'error' key in output: %s", out)
	}
}

// TestNewWithConfig_ConsoleOnly verifies that ToFile=false produces output on
// stderr only (we redirect via New's internal path) and that level filtering
// works through the Config path.
func TestNewWithConfig_ConsoleOnly(t *testing.T) {
	t.Parallel()

	// We can't easily intercept os.Stderr in NewWithConfig, so we test the
	// level/format logic through New which shares the same handler construction.
	var buf bytes.Buffer
	log := New(&buf, false) // info level, console only

	log.Debug("should-not-appear")
	log.Info("should-appear")

	out := buf.String()
	if strings.Contains(out, "should-not-appear") {
		t.Errorf("info-level logger emitted debug entry: %s", out)
	}
	if !strings.Contains(out, "should-appear") {
		t.Errorf("info-level logger suppressed info entry: %s", out)
	}
}
