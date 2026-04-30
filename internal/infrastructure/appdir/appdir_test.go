package appdir_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FeiNiaoBF/vidify/internal/infrastructure/appdir"
)

// overrideHome temporarily redirects os.UserHomeDir by setting $HOME (Unix)
// and $USERPROFILE (Windows) to a temp directory. It returns a cleanup func.
func overrideHome(t *testing.T) (string, func()) {
	t.Helper()
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	origProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmp)
	os.Setenv("USERPROFILE", tmp)
	return tmp, func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origProfile)
	}
}

// TestRoot verifies that Root returns a path containing ".vidify" and that
// the path is rooted under the current home directory.
func TestRoot(t *testing.T) {
	tmp, cleanup := overrideHome(t)
	defer cleanup()

	got, err := appdir.Root()
	if err != nil {
		t.Fatalf("Root() error = %v", err)
	}
	if !strings.Contains(got, ".vidify") {
		t.Errorf("Root() = %q; want path containing .vidify", got)
	}
	if !strings.HasPrefix(got, tmp) {
		t.Errorf("Root() = %q; want path under temp home %q", got, tmp)
	}
}

// TestConfigFile verifies the canonical config file path ends with config.yaml.
func TestConfigFile(t *testing.T) {
	_, cleanup := overrideHome(t)
	defer cleanup()

	got, err := appdir.ConfigFile()
	if err != nil {
		t.Fatalf("ConfigFile() error = %v", err)
	}
	if filepath.Base(got) != "config.yaml" {
		t.Errorf("ConfigFile() base = %q; want config.yaml", filepath.Base(got))
	}
}

// TestSubdirFunctions verifies that each sub-directory function creates its
// directory and returns the expected path suffix.
func TestSubdirFunctions(t *testing.T) {
	cases := []struct {
		name    string
		fn      func() (string, error)
		wantSub string
	}{
		{"Logs", appdir.Logs, "logs"},
		{"Models", appdir.Models, "models"},
		{"Assets", appdir.Assets, "assets"},
		{"Backgrounds", appdir.Backgrounds, "backgrounds"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, cleanup := overrideHome(t)
			defer cleanup()

			got, err := tc.fn()
			if err != nil {
				t.Fatalf("%s() error = %v", tc.name, err)
			}
			if filepath.Base(got) != tc.wantSub {
				t.Errorf("%s() = %q; want base %q", tc.name, got, tc.wantSub)
			}
			// Verify the directory was actually created.
			if fi, err := os.Stat(got); err != nil || !fi.IsDir() {
				t.Errorf("%s(): directory not created at %q", tc.name, got)
			}
		})
	}
}

// TestDefaultBackground verifies that DefaultBackground returns a path ending
// with background.jpg inside the assets directory.
func TestDefaultBackground(t *testing.T) {
	_, cleanup := overrideHome(t)
	defer cleanup()

	got, err := appdir.DefaultBackground()
	if err != nil {
		t.Fatalf("DefaultBackground() error = %v", err)
	}
	if filepath.Base(got) != "background.jpg" {
		t.Errorf("DefaultBackground() base = %q; want background.jpg", filepath.Base(got))
	}
	// The parent directory (assets/) should have been created.
	if fi, err := os.Stat(filepath.Dir(got)); err != nil || !fi.IsDir() {
		t.Errorf("DefaultBackground(): assets directory not created at %q", filepath.Dir(got))
	}
}

// TestEnsureAll verifies that EnsureAll creates all expected subdirectories
// and is idempotent (safe to call multiple times).
func TestEnsureAll(t *testing.T) {
	tmp, cleanup := overrideHome(t)
	defer cleanup()

	// First call: should create everything.
	if err := appdir.EnsureAll(); err != nil {
		t.Fatalf("EnsureAll() first call error = %v", err)
	}

	expected := []string{
		filepath.Join(tmp, ".vidify"),
		filepath.Join(tmp, ".vidify", "logs"),
		filepath.Join(tmp, ".vidify", "models"),
		filepath.Join(tmp, ".vidify", "assets"),
		filepath.Join(tmp, ".vidify", "backgrounds"),
	}
	for _, dir := range expected {
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			t.Errorf("EnsureAll(): expected directory %q to exist", dir)
		}
	}

	// Second call: must be idempotent.
	if err := appdir.EnsureAll(); err != nil {
		t.Fatalf("EnsureAll() second call (idempotent) error = %v", err)
	}
}
