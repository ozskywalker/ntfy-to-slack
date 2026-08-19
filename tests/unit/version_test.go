package unit_test

import (
	"strings"
	"testing"

	"github.com/ozskywalker/ntfy-to-slack/internal/version"
)

func TestVersionInfo_String(t *testing.T) {
	tests := []struct {
		name string
		info version.VersionInfo
		want string
	}{
		{
			name: "git tag preferred over version",
			info: version.VersionInfo{Version: "development", GitTag: "v1.2.3", GitCommit: "abcdef1234567890"},
			want: "v1.2.3 (abcdef12)",
		},
		{
			name: "version used when no git tag",
			info: version.VersionInfo{Version: "1.2.3-snapshot", GitTag: "", GitCommit: "abcdef1234567890"},
			want: "1.2.3-snapshot (abcdef12)",
		},
		{
			name: "commit shorter than 8 chars is used as-is",
			info: version.VersionInfo{Version: "development", GitCommit: "abc123"},
			want: "development (abc123)",
		},
		{
			name: "empty commit shown as unknown",
			info: version.VersionInfo{Version: "development", GitCommit: ""},
			want: "development (unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionInfo_Detailed(t *testing.T) {
	info := version.VersionInfo{
		Version:   "1.2.3",
		GitCommit: "abcdef1234567890",
		GitTag:    "v1.2.3",
		BuildDate: "2026-01-01T00:00:00Z",
		GoVersion: "go1.26.6",
		Compiler:  "gc",
		Platform:  "linux/amd64",
	}

	detailed := info.Detailed()

	for _, want := range []string{"1.2.3", "abcdef1234567890", "v1.2.3", "2026-01-01T00:00:00Z", "go1.26.6", "gc", "linux/amd64"} {
		if !strings.Contains(detailed, want) {
			t.Errorf("Detailed() = %q, missing %q", detailed, want)
		}
	}
}

func TestGet(t *testing.T) {
	info := version.Get()
	if info == nil {
		t.Fatal("Get() returned nil")
	}
	if info.GoVersion == "" {
		t.Error("Get().GoVersion is empty")
	}
	if info.Platform == "" {
		t.Error("Get().Platform is empty")
	}
}
