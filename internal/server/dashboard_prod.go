//go:build !dev

package server

import (
	"net/http"

	"github.com/larsartmann/go-health"
)

// mountHealthDashboard is the production stub. The health dashboard is a
// dev-only feature (plan M8): release binaries — Docker, goreleaser, nix,
// which all build without the dev tag — carry no dashboard routes, no SSE
// pusher, and no dashboard dependency in their link graph. The dev build
// (go build -tags dev) wires the real implementation in dashboard_dev.go.
func mountHealthDashboard(*http.ServeMux, *health.Probe) {}
