package cli

import (
	"strings"
	"testing"
	"time"
)

func TestParseAudioDuration(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    time.Duration
		wantErr string
	}{
		{
			name:    "rejects container duration without an audio stream",
			output:  `{"streams":[],"format":{"duration":"1.000000"}}`,
			wantErr: "input has no audio stream",
		},
		{
			name:   "falls back to container duration when audio stream duration is unavailable",
			output: `{"streams":[{"duration":"N/A"}],"format":{"duration":"11.000000"}}`,
			want:   11 * time.Second,
		},
		{
			name:   "prefers audio stream duration",
			output: `{"streams":[{"duration":"10.250000"}],"format":{"duration":"11.000000"}}`,
			want:   10*time.Second + 250*time.Millisecond,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseAudioDuration([]byte(test.output))
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("parseAudioDuration() error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAudioDuration() error = %v", err)
			}
			if got != test.want {
				t.Errorf("parseAudioDuration() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestParseAudioDurationRejectsMalformedJSON(t *testing.T) {
	if _, err := parseAudioDuration([]byte(`{"streams":`)); err == nil {
		t.Fatal("parseAudioDuration() error = nil, want malformed JSON error")
	}
}
