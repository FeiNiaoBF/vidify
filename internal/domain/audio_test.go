package domain_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/FeiNiaoBF/vidify/internal/domain"
)

func TestNewAudio(t *testing.T) {
	t.Parallel()

	// Create a real temp file so the existence check can pass.
	tmp, err := os.CreateTemp(t.TempDir(), "*.mp3")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmp.Close()
	absTemp, _ := filepath.Abs(tmp.Name())

	tests := []struct {
		name      string
		path      string
		wantErr   error
		wantAbs   bool // Audio.Path must be absolute
	}{
		{
			name:    "valid mp3 file",
			path:    absTemp,
			wantAbs: true,
		},
		{
			name:    "relative path is normalised to absolute",
			path:    tmp.Name(), // already absolute from os.CreateTemp; filepath.Abs is idempotent
			wantAbs: true,
		},
		{
			name:    "unsupported extension",
			path:    filepath.Join(t.TempDir(), "file.avi"),
			wantErr: domain.ErrUnsupportedFormat,
		},
		{
			name:    "supported extension but file does not exist",
			path:    filepath.Join(t.TempDir(), "ghost.mp3"),
			wantErr: domain.ErrFileNotFound,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewAudio(tt.path)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewAudio(%q): got err %v, want %v", tt.path, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NewAudio(%q): unexpected error: %v", tt.path, err)
			}
			if tt.wantAbs && !filepath.IsAbs(got.Path) {
				t.Errorf("Audio.Path is not absolute: %q", got.Path)
			}
		})
	}
}
