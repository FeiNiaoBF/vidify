package domain

import "time"

// Segment represents a single subtitle entry with timing information.
type Segment struct {
	// Index is the 1-based sequence number in the SRT file.
	Index int

	// Start is the timestamp when this segment begins.
	Start time.Duration

	// End is the timestamp when this segment ends.
	End time.Duration

	// Text is the subtitle content for this segment.
	Text string
}

// Subtitle holds the result of a Whisper transcription.
type Subtitle struct {
	// SRTPath is the filesystem path to the generated .srt file.
	SRTPath string

	// Language is the detected or specified language code (e.g. "en", "zh").
	Language string

	// Segments is the parsed list of subtitle entries.
	// May be nil if only the SRT file path is needed.
	Segments []Segment
}
