package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPaginateSegmentsKeepsSegmentsUntilPageBoundary(t *testing.T) {
	segments := make([]segment, 24)
	for index := range segments {
		start := time.Duration(index*3) * time.Second
		segments[index] = segment{
			start: start,
			end:   start + 2*time.Second,
			text:  fmt.Sprintf("Segment %02d contains a complete phrase that stays visible on this page.", index+1),
		}
	}
	audioDuration := 75 * time.Second

	pages, err := paginateSegments(segments, audioDuration)
	if err != nil {
		t.Fatalf("paginate segments: %v", err)
	}
	if len(pages) < 2 {
		t.Fatalf("expected the transcript to span multiple pages; got %d", len(pages))
	}

	eventCount := 0
	for pageIndex, page := range pages {
		if page.fontSize != initialPageFontSize {
			t.Errorf("page %d shrank its font even though no single Segment requires it: got %d px", pageIndex+1, page.fontSize)
		}
		if page.usedY > availablePageHeight() {
			t.Errorf("page %d uses %d px of %d available", pageIndex+1, page.usedY, availablePageHeight())
		}
		if len(page.events) == 0 {
			t.Fatalf("page %d has no events", pageIndex+1)
		}
		if page.start != segments[page.events[0].segmentIndex].start {
			t.Errorf("page %d starts at %s; want first segment start %s", pageIndex+1, page.start, segments[page.events[0].segmentIndex].start)
		}
		wantEnd := audioDuration
		if pageIndex+1 < len(pages) {
			wantEnd = segments[pages[pageIndex+1].events[0].segmentIndex].start
		}
		if page.end != wantEnd {
			t.Errorf("page %d ends at %s; want %s", pageIndex+1, page.end, wantEnd)
		}
		for _, event := range page.events {
			segmentText := normalizeSubtitleText(segments[event.segmentIndex].text)
			if strings.Join(event.lines, " ") != segmentText {
				t.Errorf("segment %d text changed across wrapping: got %q, want %q", event.segmentIndex+1, strings.Join(event.lines, " "), segmentText)
			}
			eventCount++
		}
	}
	if eventCount != len(segments) {
		t.Errorf("pagination retained %d of %d segments", eventCount, len(segments))
	}

	assData, err := renderASS(pages)
	if err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	srtData, err := renderSRT(segments)
	if err != nil {
		t.Fatalf("render SRT: %v", err)
	}
	first := segments[0]
	originalInterval := fmt.Sprintf("%s --> %s", srtTimestamp(first.start), srtTimestamp(first.end))
	if !strings.Contains(string(srtData), originalInterval) {
		t.Errorf("SRT does not retain original Segment interval %q", originalInterval)
	}
	if got := strings.Count(string(assData), "Dialogue: "); got != len(segments) {
		t.Errorf("ASS has %d events; want %d", got, len(segments))
	}
	for _, page := range pages {
		for _, event := range page.events {
			want := fmt.Sprintf("Dialogue: 0,%s,%s,Default", assTimestamp(event.start), assTimestamp(page.end))
			if !strings.Contains(string(assData), want) {
				t.Errorf("ASS does not keep segment %d visible through its page end %s", event.segmentIndex+1, page.end)
			}
		}
	}
}

func TestPaginateSegmentsShrinksOnlyWhenOneSegmentNeedsIt(t *testing.T) {
	pages, err := paginateSegments([]segment{{
		start: 2 * time.Second,
		end:   3 * time.Second,
		text:  strings.Repeat("compact ", 140),
	}}, 8*time.Second)
	if err != nil {
		t.Fatalf("paginate long segment: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("got %d pages; want 1", len(pages))
	}
	if pages[0].fontSize >= initialPageFontSize || pages[0].fontSize < minimumPageFontSize {
		t.Errorf("font size is %d px; want %d..%d px", pages[0].fontSize, minimumPageFontSize, initialPageFontSize-1)
	}
}

func TestPaginateSegmentsReportsUnfittableSegment(t *testing.T) {
	_, err := paginateSegments([]segment{{
		start: 2 * time.Second,
		end:   3 * time.Second,
		text:  strings.Repeat("x", 5000),
	}}, 8*time.Second)
	if err == nil {
		t.Fatal("expected an oversized segment error")
	}
	for _, want := range []string{"segment 1", "00:00:02,000", "minimum readable font size"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not contain %q", err, want)
		}
	}
}

func TestPaginateSegmentsExtendsFinalPageToAudioEnd(t *testing.T) {
	audioDuration := 10*time.Second + 375*time.Millisecond
	pages, err := paginateSegments([]segment{{
		start: 2 * time.Second,
		end:   3 * time.Second,
		text:  "A transcript segment.",
	}}, audioDuration)
	if err != nil {
		t.Fatalf("paginate segment: %v", err)
	}
	if got := pages[0].end; got != audioDuration {
		t.Errorf("final page ends at %s; want audio end %s", got, audioDuration)
	}
	assData, err := renderASS(pages)
	if err != nil {
		t.Fatalf("render ASS: %v", err)
	}
	want := fmt.Sprintf("Dialogue: 0,%s,%s,Default", assTimestamp(2*time.Second), assTimestamp(audioDuration))
	if !strings.Contains(string(assData), want) {
		t.Errorf("ASS does not extend the final Segment to audio end %s", audioDuration)
	}
}

func TestPublishSRTNoReplacePublishesCompleteFile(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "transcript.srt")
	data := []byte(strings.Repeat("1\n00:00:00,000 --> 00:00:01,000\nA complete subtitle.\n\n", 128))

	if err := publishSRTNoReplace(data, destination); err != nil {
		t.Fatalf("publish SRT: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read published SRT: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatal("published SRT content differs from the complete staged data")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Errorf("output directory contains staging residue: %v", entries)
	}
}

func TestPublishSRTNoReplaceKeepsExistingFile(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "transcript.srt")
	original := []byte("existing subtitle\n")
	if err := os.WriteFile(destination, original, 0o600); err != nil {
		t.Fatalf("create existing SRT: %v", err)
	}

	if err := publishSRTNoReplace([]byte("replacement subtitle\n"), destination); err == nil {
		t.Fatal("expected an existing-output error")
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read existing SRT: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Errorf("existing SRT changed: got %q, want %q", got, original)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(destination) {
		t.Errorf("output directory contains staging residue: %v", entries)
	}
}
