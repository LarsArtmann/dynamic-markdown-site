// Package version contains build-time version information injected via ldflags.
//
// Usage in build:
//
//	-X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${VERSION}
//	-X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${COMMIT}
//	-X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=${BUILD_DATE}
package version

// Version information injected at build time via ldflags.
//
//nolint:gochecknoglobals // These are intentionally global for ldflags injection.
var (
	// Version is the semantic version string.
	Version = "dev"

	// Commit is the git commit hash.
	Commit = "unknown"

	// BuildDate is the ISO 8601 formatted build timestamp.
	BuildDate = "unknown"
)
