//go:build dev

package server

import (
	"encoding/json/v2"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type LiveReloadEvent struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	Timestamp string `json:"timestamp"`
}

type LiveReload struct {
	logger    *slog.Logger
	mu        sync.RWMutex
	clients   map[chan LiveReloadEvent]struct{}
	closed    chan struct{}
	closeOnce sync.Once

	// heartbeat is the SSE comment cadence keeping proxies and browsers from
	// reaping idle dev streams (default 15s; shortened in tests). Comments
	// (`: heartbeat`) carry no event, so clients do not reload on them.
	heartbeat time.Duration

	// clientCap bounds concurrent SSE streams per dev server: a forgotten
	// browser tab farm would otherwise pin goroutines and file descriptors
	// forever (default 64; lowered in tests).
	clientCap int
}

const (
	defaultHeartbeatInterval = 15 * time.Second
	defaultLiveReloadClients = 64
)

func NewLiveReload(logger *slog.Logger) *LiveReload {
	return &LiveReload{
		logger:    logger,
		mu:        sync.RWMutex{},
		clients:   make(map[chan LiveReloadEvent]struct{}),
		closed:    make(chan struct{}),
		closeOnce: sync.Once{},
		heartbeat: defaultHeartbeatInterval,
		clientCap: defaultLiveReloadClients,
	}
}

// Close disconnects every SSE client and makes handlers exit immediately, so
// an open live-reload stream cannot hold http.Server.Shutdown until its
// deadline. It is idempotent and safe to call concurrently with Notify.
//
// Handlers exit through the closed signal while each handler's own
// `defer close(clientChan)` remains the only closer of its channel — closing
// them here double-closes any whose handler is still running.
func (lr *LiveReload) Close() {
	lr.closeOnce.Do(func() {
		close(lr.closed)
	})

	lr.mu.Lock()
	defer lr.mu.Unlock()

	lr.clients = make(map[chan LiveReloadEvent]struct{})
}

func (lr *LiveReload) Notify(path string) {
	event := LiveReloadEvent{
		Type:      "reload",
		Path:      path,
		Timestamp: getTimestamp(),
	}

	lr.mu.RLock()
	defer lr.mu.RUnlock()

	lr.logger.Debug("notifying live reload clients", "path", path, "client_count", len(lr.clients))

	for client := range lr.clients {
		select {
		case client <- event:
		default:
		}
	}
}

func (lr *LiveReload) handleSSE(w http.ResponseWriter, r *http.Request) {
	clientChan := make(chan LiveReloadEvent, 100)
	defer close(clientChan)

	// Cap check and registration share one critical section: two concurrent
	// connections must not both slip past a nearly-full cap (TOCTOU).
	lr.mu.Lock()
	if len(lr.clients) >= lr.clientCap {
		lr.mu.Unlock()

		lr.logger.Warn(
			"live reload client rejected: cap reached",
			"client_ip", clientIP(r),
			"cap", lr.clientCap,
		)

		http.Error(
			w,
			fmt.Sprintf("live reload client limit reached (%d); close other tabs and reload", lr.clientCap),
			http.StatusServiceUnavailable,
		)

		return
	}

	lr.clients[clientChan] = struct{}{}
	clientCount := len(lr.clients)
	lr.mu.Unlock()

	defer func() {
		lr.mu.Lock()
		delete(lr.clients, clientChan)
		lr.mu.Unlock()
	}()

	lr.logger.Info(
		"live reload client connected",
		"client_ip", clientIP(r),
		"total_clients", clientCount,
	)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)

		return
	}

	w.Header().Set(headerContentType, "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	connectedData, err := json.Marshal(map[string]string{jsonKeyMessage: "connected to live reload"})
	if err != nil {
		lr.logger.Error("failed to marshal connected event", "error", err)

		return
	}

	_, _ = fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connectedData) //nolint:erraudit // disconnect unrecoverable
	flusher.Flush()

	ctx := r.Context()

	// The ticker lives on the handler loop: heartbeats and reload events are
	// written by the same goroutine, so frames never interleave.
	heartbeat := time.NewTicker(lr.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			lr.logger.Debug("live reload client disconnected")

			return
		case <-lr.closed:
			lr.logger.Debug("live reload closed, disconnecting client")

			return
		case <-heartbeat.C:
			// SSE comment frame: invisible to event listeners, resets idle
			// timers in browsers and intermediate proxies.
			if _, err := fmt.Fprint(w, ": heartbeat\n\n"); err != nil {
				lr.logger.Debug("failed to send live reload heartbeat", "error", err)

				return
			}

			flusher.Flush()
		case event, ok := <-clientChan:
			if !ok {
				return
			}

			data, err := json.Marshal(event)
			if err != nil {
				lr.logger.Error("failed to marshal live reload event", "error", err)

				continue
			}

			if _, err := fmt.Fprintf(w, "event: reload\ndata: %s\n\n", data); err != nil {
				lr.logger.Debug("failed to send live reload event", "error", err)
			}

			flusher.Flush()
		}
	}
}

func getTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// registerLiveReloadRoute mounts the SSE endpoint when running a dev-mode
// server. It exists so handlers.go needs no build-tag knowledge; the prod
// variant is a no-op because the code path does not exist there at all.
func (s *Server) registerLiveReloadRoute(mux *http.ServeMux) {
	if s.devMode {
		mux.HandleFunc("GET /api/live-reload", s.liveReload.handleSSE)
	}
}
