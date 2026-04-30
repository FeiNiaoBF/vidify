package domain

// WhisperModel enumerates the available Whisper model sizes.
// Smaller models are faster but less accurate; larger models need more RAM.
type WhisperModel string

const (
	ModelTiny   WhisperModel = "tiny"
	ModelBase   WhisperModel = "base"
	ModelSmall  WhisperModel = "small"
	ModelMedium WhisperModel = "medium"
	ModelLarge  WhisperModel = "large"
)

// validModels is used for O(1) validation of user-supplied model names.
var validModels = map[WhisperModel]bool{
	ModelTiny:   true,
	ModelBase:   true,
	ModelSmall:  true,
	ModelMedium: true,
	ModelLarge:  true,
}

// IsValidModel checks whether a model string is a known Whisper model.
func IsValidModel(m string) bool {
	return validModels[WhisperModel(m)]
}

// Config holds all user-supplied parameters for a single processing task.
// It is constructed from CLI flags / config file and passed through layers.
type Config struct {
	// InputPath is the audio file or directory to process.
	InputPath string

	// OutputDir is the directory where output files will be written.
	// Defaults to the current working directory if empty.
	OutputDir string

	// Model specifies which Whisper model to use for transcription.
	Model WhisperModel

	// Language is the BCP-47 language code (e.g. "en", "zh", "ja").
	// Empty string means auto-detect.
	Language string

	// Translate enables Whisper's translation mode (non-English → English).
	Translate bool

	// BGImagePath is the path to a background image for the video.
	// Empty string means use the built-in default.
	BGImagePath string

	// WhisperBin is the path to the whisper-cli executable.
	WhisperBin string

	// WhisperModelDir is the directory containing Whisper model files.
	WhisperModelDir string

	// FFmpegBin is the path to the ffmpeg executable.
	FFmpegBin string

	// Verbose enables debug-level logging output.
	Verbose bool
}

// Validate checks that required fields are present and values are valid.
func (c *Config) Validate() error {
	if c.InputPath == "" {
		return ErrMissingInput
	}
	if !IsValidModel(string(c.Model)) {
		return ErrInvalidModel
	}
	return nil
}
