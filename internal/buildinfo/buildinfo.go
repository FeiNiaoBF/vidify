package buildinfo

import "fmt"

// These values can be overridden at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Summary returns a single-line build summary useful for CLI output.
func Summary() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
