package version

import (
	"strings"
	"testing"
)

func TestBuildInfo(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	Version = "1.0.0"
	Commit = "abc123"
	Date = "2026-01-01T00:00:00Z"

	info := BuildInfo()
	if !strings.Contains(info, "version=1.0.0") {
		t.Errorf("BuildInfo() = %q, want contains %q", info, "version=1.0.0")
	}
	if !strings.Contains(info, "commit=abc123") {
		t.Errorf("BuildInfo() = %q, want contains %q", info, "commit=abc123")
	}
	if !strings.Contains(info, "date=2026-01-01T00:00:00Z") {
		t.Errorf("BuildInfo() = %q, want contains %q", info, "date=2026-01-01T00:00:00Z")
	}
}

func TestBuildInfo_DevValues(t *testing.T) {
	// Save original values
	origVersion := Version
	origCommit := Commit
	origDate := Date
	defer func() {
		Version = origVersion
		Commit = origCommit
		Date = origDate
	}()

	Version = "dev"
	Commit = "dev"
	Date = "unknown"

	info := BuildInfo()
	expected := "version=dev commit=dev date=unknown"
	if info != expected {
		t.Errorf("BuildInfo() = %q, want %q", info, expected)
	}
}

func TestShort(t *testing.T) {
	// Save original value
	origVersion := Version
	defer func() { Version = origVersion }()

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"normal version", "1.0.0", "1.0.0"},
		{"dev version", "dev", "dev"},
		{"empty version", "", "dev"},
		{"whitespace version", "  ", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			got := Short()
			if got != tt.want {
				t.Errorf("Short() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShort_EmptyVersion(t *testing.T) {
	// Save original value
	origVersion := Version
	defer func() { Version = origVersion }()

	Version = ""
	if got := Short(); got != "dev" {
		t.Errorf("Short() = %q, want %q", got, "dev")
	}
}
