package version

import "strings"

var (
	// Version is set at build time via -ldflags, defaulting to the current release.
	Version = "0.1.2"
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
