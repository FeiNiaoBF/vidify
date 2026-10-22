package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type subtitlePage struct {
	start      time.Duration
	end        time.Duration
	fontSize   int
	lineHeight int
	events     []subtitleEvent
	usedY      int
}

type subtitleEvent struct {
	segmentIndex int
	start        time.Duration
	y            int
	lines        []string
}

// DejaVu Serif is a provisional font; embedding it keeps the measured and
// libass-rendered glyph shapes consistent across supported hosts.
//
//go:embed assets/DejaVuSerif.ttf
var provisionalSerifFont []byte

// These provisional layout values keep measurement and rendering deterministic;
// final margins, font size, and visual style remain open in the MVP ADR.
const (
	videoWidth          = 1920
	videoHeight         = 1080
	pageMarginLeft      = 120
	pageMarginRight     = 120
	pageMarginTop       = 100
	pageMarginBottom    = 100
	initialPageFontSize = 48
	minimumPageFontSize = 32
	assOutlineSize      = 2
)

var errGlyphTooWide = errors.New("a glyph is wider than the available subtitle area")

type segmentLayout struct {
	lines      []string
	height     int
	lineHeight int
}

type pageLayouter struct {
	font  *opentype.Font
	faces map[int]font.Face
}

func paginateSegments(segments []segment, audioDuration time.Duration) ([]subtitlePage, error) {
	if len(segments) == 0 {
		return nil, errors.New("cannot paginate an empty transcript")
	}
	if audioDuration <= 0 {
		return nil, errors.New("audio duration must be positive for subtitle pagination")
	}
	fontFile, err := opentype.Parse(provisionalSerifFont)
	if err != nil {
		return nil, fmt.Errorf("parse embedded provisional font: %w", err)
	}
	layouter := &pageLayouter{font: fontFile, faces: make(map[int]font.Face)}
	defer layouter.close()

	pages := make([]*subtitlePage, 0)
	var page *subtitlePage
	for index, item := range segments {
		if item.start >= audioDuration {
			return nil, segmentLayoutError(index, item.start, "segment starts at or after the audio ends")
		}
		if page != nil {
			layout, err := layouter.layout(item.text, page.fontSize)
			if err != nil && !errors.Is(err, errGlyphTooWide) {
				return nil, segmentLayoutError(index, item.start, err.Error())
			}
			if err == nil && pageCanFit(page, layout) {
				addSegmentToPage(page, index, item, layout)
				continue
			}
			if item.start <= page.start {
				return nil, segmentLayoutError(index, item.start, "a page break is required before this segment, but the current page has no display time")
			}
			page.end = item.start
			page = nil
		}

		fontSize := initialPageFontSize
		var layout segmentLayout
		var layoutErr error
		for ; fontSize >= minimumPageFontSize; fontSize-- {
			layout, layoutErr = layouter.layout(item.text, fontSize)
			if layoutErr != nil {
				if errors.Is(layoutErr, errGlyphTooWide) {
					continue
				}
				return nil, segmentLayoutError(index, item.start, layoutErr.Error())
			}
			if layout.height <= availablePageHeight() {
				break
			}
			layoutErr = fmt.Errorf("text needs %d vertical pixels, page has %d", layout.height, availablePageHeight())
		}
		if fontSize < minimumPageFontSize {
			if layoutErr == nil {
				layoutErr = errors.New("text exceeds the available area at the minimum font size")
			}
			return nil, segmentLayoutError(index, item.start, fmt.Sprintf("does not fit at the minimum readable font size (%d px): %v", minimumPageFontSize, layoutErr))
		}
		page = &subtitlePage{start: item.start, fontSize: fontSize, lineHeight: layout.lineHeight}
		addSegmentToPage(page, index, item, layout)
		pages = append(pages, page)
	}
	if page == nil {
		return nil, errors.New("pagination produced no final page")
	}
	page.end = audioDuration

	result := make([]subtitlePage, len(pages))
	for index, page := range pages {
		result[index] = *page
	}
	return result, nil
}

func (l *pageLayouter) close() {
	for _, face := range l.faces {
		_ = face.Close()
	}
}

func (l *pageLayouter) layout(text string, size int) (segmentLayout, error) {
	face, ok := l.faces[size]
	if !ok {
		var err error
		face, err = opentype.NewFace(l.font, &opentype.FaceOptions{
			Size:    float64(size),
			DPI:     72,
			Hinting: font.HintingNone,
		})
		if err != nil {
			return segmentLayout{}, fmt.Errorf("create %d px font face: %w", size, err)
		}
		l.faces[size] = face
	}
	lineHeight := face.Metrics().Height.Ceil() + 2*assOutlineSize
	if lineHeight <= 0 {
		return segmentLayout{}, fmt.Errorf("font has invalid line height at %d px", size)
	}
	lines, err := wrapSegmentText(face, text, videoWidth-pageMarginLeft-pageMarginRight)
	if err != nil {
		return segmentLayout{}, err
	}
	return segmentLayout{lines: lines, height: len(lines) * lineHeight, lineHeight: lineHeight}, nil
}

func wrapSegmentText(face font.Face, text string, maxWidth int) ([]string, error) {
	words := strings.Fields(normalizeSubtitleText(text))
	if len(words) == 0 {
		return nil, errors.New("segment text is empty after whitespace normalization")
	}

	lines := make([]string, 0, len(words))
	line := ""
	for _, word := range words {
		candidate := word
		if line != "" {
			candidate = line + " " + word
		}
		width, err := measuredTextWidth(face, candidate)
		if err != nil {
			return nil, err
		}
		if width <= maxWidth {
			line = candidate
			continue
		}
		if line != "" {
			lines = append(lines, line)
			line = ""
		}
		chunks, err := wrapLongWord(face, word, maxWidth)
		if err != nil {
			return nil, err
		}
		lines = append(lines, chunks[:len(chunks)-1]...)
		line = chunks[len(chunks)-1]
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines, nil
}

func wrapLongWord(face font.Face, word string, maxWidth int) ([]string, error) {
	chunks := make([]string, 0, 1)
	chunk := ""
	for _, character := range word {
		candidate := chunk + string(character)
		width, err := measuredTextWidth(face, candidate)
		if err != nil {
			return nil, err
		}
		if width > maxWidth {
			if chunk == "" {
				return nil, fmt.Errorf("%w: U+%04X exceeds %d px", errGlyphTooWide, character, maxWidth)
			}
			chunks = append(chunks, chunk)
			chunk = string(character)
			width, err = measuredTextWidth(face, chunk)
			if err != nil {
				return nil, err
			}
			if width > maxWidth {
				return nil, fmt.Errorf("%w: U+%04X exceeds %d px", errGlyphTooWide, character, maxWidth)
			}
			continue
		}
		chunk = candidate
	}
	if chunk != "" {
		chunks = append(chunks, chunk)
	}
	return chunks, nil
}

func measuredTextWidth(face font.Face, text string) (int, error) {
	for _, character := range text {
		if _, ok := face.GlyphAdvance(character); !ok {
			return 0, fmt.Errorf("embedded provisional font has no glyph for U+%04X", character)
		}
	}
	bounds, advance := font.BoundString(face, text)
	left := min(0, bounds.Min.X.Floor())
	right := max(advance.Ceil(), bounds.Max.X.Ceil())
	return right - left + 2*assOutlineSize, nil
}

func pageCanFit(page *subtitlePage, layout segmentLayout) bool {
	additionalHeight := layout.height
	if len(page.events) > 0 {
		additionalHeight += segmentGap(page.lineHeight)
	}
	return page.usedY+additionalHeight <= availablePageHeight()
}

func addSegmentToPage(page *subtitlePage, index int, item segment, layout segmentLayout) {
	if len(page.events) > 0 {
		page.usedY += segmentGap(page.lineHeight)
	}
	page.events = append(page.events, subtitleEvent{
		segmentIndex: index,
		start:        item.start,
		y:            pageMarginTop + page.usedY,
		lines:        layout.lines,
	})
	page.usedY += layout.height
}

func availablePageHeight() int {
	return videoHeight - pageMarginTop - pageMarginBottom
}

func segmentGap(lineHeight int) int {
	return max(8, lineHeight/3)
}

func segmentLayoutError(index int, start time.Duration, reason string) error {
	return fmt.Errorf("cannot lay out segment %d at %s: %s", index+1, srtTimestamp(start), reason)
}

func renderASS(pages []subtitlePage) ([]byte, error) {
	if len(pages) == 0 {
		return nil, errors.New("cannot write ASS without subtitle pages")
	}
	var output strings.Builder
	output.WriteString("[Script Info]\nTitle: Vidify transcript\nScriptType: v4.00+\nPlayResX: 1920\nPlayResY: 1080\nWrapStyle: 2\nCollisions: Normal\nScaledBorderAndShadow: yes\n\n")
	output.WriteString("[V4+ Styles]\nFormat: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding\n")
	fmt.Fprintf(&output, "Style: Default,DejaVu Serif,%d,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,%d,0,7,%d,%d,%d,1\n\n", initialPageFontSize, assOutlineSize, pageMarginLeft, pageMarginRight, pageMarginTop)
	output.WriteString("[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n")
	for _, page := range pages {
		if page.end <= page.start {
			return nil, fmt.Errorf("invalid subtitle page interval %s..%s", srtTimestamp(page.start), srtTimestamp(page.end))
		}
		for _, event := range page.events {
			start := assTimestamp(event.start)
			end := assTimestamp(page.end)
			if page.end <= event.start || end <= start {
				return nil, segmentLayoutError(event.segmentIndex, event.start, "ASS centisecond precision cannot represent a positive display interval")
			}
			lines := make([]string, len(event.lines))
			for index, line := range event.lines {
				lines[index] = escapeASSText(line)
			}
			text := strings.Join(lines, `\N`)
			fmt.Fprintf(&output, "Dialogue: 0,%s,%s,Default,,0,0,0,,{\\an7\\pos(%d,%d)\\fs%d}%s\n", start, end, pageMarginLeft, event.y, page.fontSize, text)
		}
	}
	return []byte(output.String()), nil
}

func assTimestamp(value time.Duration) string {
	centiseconds := int64(value.Round(10*time.Millisecond) / (10 * time.Millisecond))
	hours := centiseconds / 360_000
	minutes := centiseconds / 6_000 % 60
	seconds := centiseconds / 100 % 60
	fraction := centiseconds % 100
	return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, fraction)
}

func writeProvisionalFont(workDir string) error {
	fontDir := filepath.Join(workDir, "fonts")
	if err := os.Mkdir(fontDir, 0o700); err != nil {
		return fmt.Errorf("create temporary font directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(fontDir, "DejaVuSerif.ttf"), provisionalSerifFont, 0o600); err != nil {
		return fmt.Errorf("stage embedded provisional font for libass: %w", err)
	}
	return nil
}

func escapeASSText(value string) string {
	value = normalizeSubtitleText(value)
	value = strings.ReplaceAll(value, "{", `\{`)
	return strings.ReplaceAll(value, "}", `\}`)
}
