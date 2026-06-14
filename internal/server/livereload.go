package server

import (
	"encoding/json"
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
	logger  *slog.Logger
	mu      sync.RWMutex
	clients map[chan LiveReloadEvent]struct{}
}

func NewLiveReload(logger *slog.Logger) *LiveReload {
	return &LiveReload{
		logger:  logger,
		mu:      sync.RWMutex{},
		clients: make(map[chan LiveReloadEvent]struct{}),
	}
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

	lr.mu.Lock()
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

	_, _ = fmt.Fprintf(w, "event: connected\ndata: %s\n\n", connectedData)
	flusher.Flush()

	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			lr.logger.Debug("live reload client disconnected")

			return
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
