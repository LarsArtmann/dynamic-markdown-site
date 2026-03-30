package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// LiveReloadEvent represents an event sent to connected clients.
type LiveReloadEvent struct {
	Type      string `json:"type"`
	Path      string `json:"path,omitempty"`
	Timestamp string `json:"timestamp"`
}

// LiveReload manages Server-Sent Events connections for live reload functionality.
type LiveReload struct {
	logger *slog.Logger
	mu     sync.RWMutex
	// clients holds all connected SSE clients
	clients map[chan LiveReloadEvent]struct{}
}

// NewLiveReload creates a new LiveReload manager.
func NewLiveReload(logger *slog.Logger) *LiveReload {
	return &LiveReload{
		logger:  logger,
		mu:      sync.RWMutex{},
		clients: make(map[chan LiveReloadEvent]struct{}),
	}
}

// RegisterHandler registers the SSE endpoint for live reload.
func (lr *LiveReload) RegisterHandler(router *gin.Engine) {
	router.GET("/api/live-reload", lr.handleSSE)
}

// Notify sends a reload event to all connected clients.
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
			// Channel full, skip this client
		}
	}
}

func (lr *LiveReload) handleSSE(c *gin.Context) {
	// Create buffered channel for this client
	clientChan := make(chan LiveReloadEvent, 100)
	defer close(clientChan)

	// Register client
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
		"client_ip",
		c.ClientIP(),
		"total_clients",
		clientCount,
	)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Send initial connection event
	c.SSEvent("connected", gin.H{
		"message": "connected to live reload",
	})
	c.Writer.Flush()

	// Stream events to client
	clientGone := c.Request.Context().Done()
	for {
		select {
		case <-clientGone:
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
			if _, err := fmt.Fprintf(c.Writer, "event: reload\ndata: %s\n\n", data); err != nil {
				lr.logger.Debug("failed to send live reload event", "error", err)
			}
			c.Writer.Flush()
		}
	}
}

// getTimestamp returns the current Unix timestamp as a string.
func getTimestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}
