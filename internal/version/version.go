// Package version contains build-time version information injected via ldflags.
//
// Usage in build:
//   -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}
package version

// Version information injected at build time via ldflags.
var (
	// Version is the semantic version string.
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// BuildDate is the ISO 8601 formatted build timestamp.
	BuildDate = "unknown"
)
