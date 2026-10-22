package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type whisperJSON struct {
	Result struct {
		Language string `json:"language"`
	} `json:"result"`
	Transcription []struct {
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
		Text string `json:"text"`
	} `json:"transcription"`
}

type segment struct {
	start time.Duration
	end   time.Duration
	text  string
}

func readWhisperSegments(path string) ([]segment, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read whisper.cpp JSON output: %w", err)
	}
	var output whisperJSON
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("parse whisper.cpp JSON output: %w", err)
	}
	if len(output.Transcription) == 0 {
		return nil, errors.New("whisper.cpp returned no transcription segments")
	}
	segments := make([]segment, 0, len(output.Transcription))
	var previousStart time.Duration = -1
	for index, item := range output.Transcription {
		start := time.Duration(item.Offsets.From) * time.Millisecond
		end := time.Duration(item.Offsets.To) * time.Millisecond
		text := strings.TrimSpace(item.Text)
		if start < 0 || end <= start {
			return nil, fmt.Errorf("invalid timestamp interval in segment %d: %d..%d ms", index+1, item.Offsets.From, item.Offsets.To)
		}
		if start < previousStart {
			return nil, fmt.Errorf("transcription timestamps are out of order at segment %d", index+1)
		}
		if text == "" {
			return nil, fmt.Errorf("transcription segment %d has empty text", index+1)
		}
		segments = append(segments, segment{start: start, end: end, text: text})
		previousStart = start
	}
	return segments, nil
}

func renderSRT(segments []segment) ([]byte, error) {
	if len(segments) == 0 {
		return nil, errors.New("cannot write SRT without transcription segments")
	}
	var output strings.Builder
	for index, item := range segments {
		if _, err := fmt.Fprintf(&output, "%d\n%s --> %s\n%s\n\n", index+1, srtTimestamp(item.start), srtTimestamp(item.end), normalizeSubtitleText(item.text)); err != nil {
			return nil, fmt.Errorf("format SRT segment %d: %w", index+1, err)
		}
	}
	return []byte(output.String()), nil
}

func srtTimestamp(value time.Duration) string {
	milliseconds := value.Milliseconds()
	hours := milliseconds / 3_600_000
	minutes := milliseconds / 60_000 % 60
	seconds := milliseconds / 1_000 % 60
	fraction := milliseconds % 1_000
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, fraction)
}

func normalizeSubtitleText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.ReplaceAll(value, "\n", " ")
}
