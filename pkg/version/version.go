package version

import (
	"fmt"
	"runtime"
)

// Build-time variables injected via -ldflags.
var (
	// Version is the current semantic version of NewScid.
	Version = "1.3.19"

	// GitCommit is the git commit hash at build time.
	GitCommit = "development"

	// BuildDate is the UTC ISO8601 timestamp at build time.
	BuildDate = "unknown"

	// Channel is the release stream (e.g., stable, beta, dev).
	Channel = "stable"
)

// Info holds complete version and runtime metadata.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildDate string `json:"buildDate"`
	Channel   string `json:"channel"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
	Compiler  string `json:"compiler"`
}

// Get returns the structured version information.
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		Channel:   Channel,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Compiler:  runtime.Compiler,
	}
}

// String returns a single-line human readable summary.
func String() string {
	return fmt.Sprintf("NewScid v%s (commit: %s, built: %s, %s/%s)",
		Version, GitCommit, BuildDate, runtime.GOOS, runtime.GOARCH)
}

// Short returns the version string with 'v' prefix.
func Short() string {
	return fmt.Sprintf("v%s", Version)
}
