package version

import (
	"fmt"
	"runtime"
)

// Build-time variables set via ldflags
var (
	Version   = "development"
	GitCommit = ""
	BuildDate = "1970-01-01T00:00:00Z"
	GitTag    = ""
)

// VersionInfo contains comprehensive version information
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	GitTag    string `json:"gitTag"`
	BuildDate string `json:"buildDate"`
	GoVersion string `json:"goVersion"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Get returns comprehensive version information
func Get() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		GitTag:    GitTag,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string suitable for display
func (v *VersionInfo) String() string {
	commitShort := v.GitCommit
	if len(commitShort) > 8 {
		commitShort = commitShort[:8]
	}
	if commitShort == "" {
		commitShort = "unknown"
	}

	if v.GitTag != "" {
		return fmt.Sprintf("%s (%s)", v.GitTag, commitShort)
	}
	if v.Version != "development" {
		return fmt.Sprintf("%s (%s)", v.Version, commitShort)
	}
	return fmt.Sprintf("%s (%s)", v.Version, commitShort)
}

// Detailed returns detailed version information for verbose output
func (v *VersionInfo) Detailed() string {
	return fmt.Sprintf(
		"Version:    %s\nGit Commit: %s\nGit Tag:    %s\nBuild Date: %s\nGo Version: %s\nCompiler:   %s\nPlatform:   %s",
		v.Version,
		v.GitCommit,
		v.GitTag,
		v.BuildDate,
		v.GoVersion,
		v.Compiler,
		v.Platform,
	)
}
