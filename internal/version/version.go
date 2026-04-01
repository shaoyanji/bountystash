package version

import "strings"

var (
	// Version is set at build time via -ldflags.
	// Default value is a placeholder - actual version comes from:
	//   - Nix flake builds: reads VERSION file
	//   - Go builds: use -ldflags "-X github.com/shaoyanji/bountystash/internal/version.Version=X.Y.Z"
	//   - task build:web / task build:tui: auto-injects from VERSION file
	Version = "dev"
	// Commit is set at build time via -ldflags.
	Commit = "dev"
	// Date is set at build time via -ldflags (UTC RFC3339 is recommended).
	Date = "unknown"
)

func BuildInfo() string {
	return "version=" + Version + " commit=" + Commit + " date=" + Date
}

func Short() string {
	if strings.TrimSpace(Version) == "" {
		return "dev"
	}
	return Version
}
