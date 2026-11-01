//go:build !dev

package server

import (
	"log/slog"
	"net/http"
)

// LiveReload is the production stub: compiled out of release binaries, so no
// SSE code path, hub, or route exists outside dev builds (plan decision Q3).
// The dev build (go build -tags dev) carries the real implementation in
// livereload_dev.go.
type LiveReload struct{}

func NewLiveReload(_ *slog.Logger) *LiveReload { return &LiveReload{} }

// Close is a no-op; the stub holds no resources.
func (lr *LiveReload) Close() {}

// Notify is a no-op; the dev-mode file watcher may still call it when a
// prod-built binary runs with -dev, and there is nothing to notify.
func (lr *LiveReload) Notify(string) {}

func (s *Server) registerLiveReloadRoute(*http.ServeMux) {}
