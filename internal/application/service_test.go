package application

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/FeiNiaoBF/vidify/internal/application/usecase"
	"github.com/FeiNiaoBF/vidify/internal/domain"
	"github.com/FeiNiaoBF/vidify/internal/domain/port"
)

// ---- mock helpers ----------------------------------------------------------

// spyLogger records every call made through port.Logger so tests can assert
// that specific levels and messages were logged.
type spyLogger struct {
	mu      sync.Mutex
	entries []spyEntry
}

type spyEntry struct {
	level string
	msg   string
}

func (s *spyLogger) record(level, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, spyEntry{level: level, msg: msg})
}

func (s *spyLogger) Debug(msg string, _ ...any)    { s.record("debug", msg) }
func (s *spyLogger) Info(msg string, _ ...any)     { s.record("info", msg) }
func (s *spyLogger) Warn(msg string, _ ...any)     { s.record("warn", msg) }
func (s *spyLogger) Error(msg string, _ ...any)    { s.record("error", msg) }
func (s *spyLogger) With(_ ...any) port.Logger     { return s }
func (s *spyLogger) WithError(_ error) port.Logger { return s }

// hasEntry returns true if any logged entry matches the given level and
// contains msgSubstr somewhere in the message string.
func (s *spyLogger) hasEntry(level, msgSubstr string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.entries {
		if e.level == level && containsStr(e.msg, msgSubstr) {
			return true
		}
	}
	return false
}

func containsStr(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// stubTranscriber returns configurable results for the port.Transcriber port.
type stubTranscriber struct {
	subtitle domain.Subtitle
	err      error
}

func (s *stubTranscriber) Transcribe(_ context.Context, _ domain.Audio, _ domain.Config) (domain.Subtitle, error) {
	return s.subtitle, s.err
}

// stubMerger returns configurable results for the port.VideoMerger port.
type stubMerger struct {
	video domain.Video
	err   error
}

func (s *stubMerger) Merge(_ context.Context, _ domain.Audio, _ domain.Subtitle, _ domain.Config) (domain.Video, error) {
	return s.video, s.err
}

// stubProgress is a no-op port.ProgressReporter.
type stubProgress struct{}

func (s *stubProgress) Start(_ string, _ int64) {}
func (s *stubProgress) Update(_ int64)          {}
func (s *stubProgress) Finish()                 {}
func (s *stubProgress) Wait()                   {}

// newTestService wires a Service with a spy logger and the given driven stubs.
func newTestService(t *testing.T, spy *spyLogger, tr port.Transcriber, mr port.VideoMerger) *Service {
	t.Helper()
	tc := usecase.NewTranscribe(tr, &stubProgress{}, spy)
	rc := usecase.NewRender(mr, &stubProgress{}, spy)
	// Pass nil taskDirFn — tests do not need a persistent cache directory.
	return NewService(tc, rc, spy, nil)
}

// ---- tests -----------------------------------------------------------------

// TestProcessFile_InvalidInput checks that an unsupported / missing input path
// is logged at Warn level and that the returned Result carries an error.
func TestProcessFile_InvalidInput(t *testing.T) {
	t.Parallel()

	spy := &spyLogger{}
	svc := newTestService(t, spy, &stubTranscriber{}, &stubMerger{})

	result := svc.ProcessFile(context.Background(), domain.Config{
		InputPath: "/no/such/file.xyz",
	})

	if result.Err == nil {
		t.Fatal("expected error for unsupported input, got nil")
	}
	if !spy.hasEntry("warn", "invalid input") {
		t.Errorf("expected warn log for invalid input; entries: %+v", spy.entries)
	}
}

// TestProcessFile_HappyPath verifies that a successful pipeline logs start and
// completion messages and returns a non-empty OutputPath.
func TestProcessFile_HappyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	audioPath := filepath.Join(dir, "test.mp3")
	// Create a minimal dummy file so domain.NewAudio's Stat check passes.
	if err := os.WriteFile(audioPath, []byte{}, 0o644); err != nil {
		t.Fatalf("cannot create temp audio file: %v", err)
	}

	outPath := filepath.Join(dir, "out.mp4")
	spy := &spyLogger{}
	svc := newTestService(t, spy,
		&stubTranscriber{subtitle: domain.Subtitle{SRTPath: filepath.Join(dir, "sub.srt")}},
		&stubMerger{video: domain.Video{OutputPath: outPath}},
	)

	result := svc.ProcessFile(context.Background(), domain.Config{
		InputPath: audioPath,
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.OutputPath == "" {
		t.Error("expected non-empty OutputPath")
	}
	if !spy.hasEntry("info", "starting audio-to-video processing") {
		t.Errorf("expected start log; entries: %+v", spy.entries)
	}
	if !spy.hasEntry("info", "processing complete") {
		t.Errorf("expected completion log; entries: %+v", spy.entries)
	}
}
