package domain

// Video holds metadata about a generated MP4 video.
type Video struct {
	// OutputPath is the filesystem path to the final MP4 file.
	OutputPath string

	// DurationSec is the video duration in seconds.
	DurationSec float64
}
