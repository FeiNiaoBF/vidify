package ffmpeg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSubtitleFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		srtPath string
		// wantContains checks that the filter script file contains this substring.
		wantContains string
	}{
		{
			name:         "plain unix path",
			srtPath:      "/tmp/output.srt",
			wantContains: "/tmp/output.srt",
		},
		{
			name:         "path with single quote",
			srtPath:      "/home/user/it's a podcast.srt",
			wantContains: `it\'s a podcast`,
		},
		{
			name:         "windows path with backslash",
			srtPath:      `C:\Users\alice\output.srt`,
			wantContains: "C:/Users/alice/output.srt",
		},
		{
			name:         "path with comma",
			srtPath:      "/tmp/ep,1.srt",
			wantContains: "ep,1.srt", // commas are safe inside script file
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			scriptPath, cleanup, err := buildSubtitleFilter(tt.srtPath, "")
			if err != nil {
				t.Fatalf("buildSubtitleFilter(%q): unexpected error: %v", tt.srtPath, err)
			}
			defer cleanup()

			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("read script file: %v", err)
			}

			if !strings.Contains(string(content), tt.wantContains) {
				t.Errorf("script file content:\n%s\ndoes not contain %q", content, tt.wantContains)
			}
		})
	}
}

func TestSafeOutputPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	base := "podcast"
	ext := ".mp4"

	// No conflict: should return the plain path.
	got := safeOutputPath(dir, base, ext)
	want := filepath.Join(dir, base+ext)
	if got != want {
		t.Errorf("no conflict: got %q, want %q", got, want)
	}

	// Create the file to force a conflict.
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatalf("create conflict file: %v", err)
	}

	// First conflict: should return podcast_1.mp4.
	got = safeOutputPath(dir, base, ext)
	want1 := filepath.Join(dir, "podcast_1.mp4")
	if got != want1 {
		t.Errorf("one conflict: got %q, want %q", got, want1)
	}

	// Create _1 as well.
	if err := os.WriteFile(want1, []byte("x"), 0o644); err != nil {
		t.Fatalf("create conflict_1 file: %v", err)
	}

	// Second conflict: should return podcast_2.mp4.
	got = safeOutputPath(dir, base, ext)
	want2 := filepath.Join(dir, "podcast_2.mp4")
	if got != want2 {
		t.Errorf("two conflicts: got %q, want %q", got, want2)
	}
}
