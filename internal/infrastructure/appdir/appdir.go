// Package appdir is the single authority for all ~/.vidify/* path lookups.
//
// Every infrastructure adapter that needs a default path under the application
// data directory should call functions from this package rather than
// constructing paths inline. This keeps the directory layout in one place and
// makes it easy to change or override for tests.
package appdir

import (
	"fmt"
	"os"
	"path/filepath"
)

// Root returns the root application data directory: ~/.vidify.
// The directory is NOT created by this function; use EnsureAll or subdir
// functions for that.
func Root() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("appdir: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".vidify"), nil
}

// ConfigFile returns the canonical path to the primary config file
// (~/.vidify/config.yaml). It does not create the file or its parent
// directory; that is the responsibility of `vidify init`.
func ConfigFile() (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config.yaml"), nil
}

// Logs returns the path to the log directory (~/.vidify/logs),
// creating it if it does not already exist.
func Logs() (string, error) {
	return subdir("logs")
}

// Models returns the path to the model directory (~/.vidify/models),
// creating it if it does not already exist.
func Models() (string, error) {
	return subdir("models")
}

// Assets returns the path to the built-in assets directory
// (~/.vidify/assets), creating it if it does not already exist.
func Assets() (string, error) {
	return subdir("assets")
}

// Backgrounds returns the path to the user backgrounds directory
// (~/.vidify/backgrounds), creating it if it does not already exist.
func Backgrounds() (string, error) {
	return subdir("backgrounds")
}

// DefaultBackground returns the expected path of the default background
// image (~/.vidify/assets/background.jpg). The file may not exist yet;
// callers must check with os.Stat before attempting to use it.
func DefaultBackground() (string, error) {
	assets, err := Assets()
	if err != nil {
		return "", err
	}
	return filepath.Join(assets, "background.jpg"), nil
}

// EnsureAll creates every standard subdirectory under ~/.vidify.
// It is idempotent: calling it multiple times has the same effect as
// calling it once. A partial failure returns the first error encountered
// but may have already created some directories.
func EnsureAll() error {
	root, err := Root()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("appdir: create root directory: %w", err)
	}
	for _, name := range []string{"logs", "models", "assets", "backgrounds"} {
		if _, err := subdir(name); err != nil {
			return err
		}
	}
	return nil
}

// subdir creates a named directory under Root and returns its path.
// It is the canonical way to obtain a writable subdirectory path; all
// exported sub-directory helpers delegate to it.
func subdir(name string) (string, error) {
	root, err := Root()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("appdir: create %s directory: %w", name, err)
	}
	return dir, nil
}
