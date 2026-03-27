// Cyberdom Site Generator - Main entry point
//
// A type-safe, high-performance markdown-to-website converter with
// a distinctive Cyberdom aesthetic.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cockroachdberrors "github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/container"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
)

// Server timeouts.
const (
	// idleTimeout is how long to wait for idle connections before closing.
	idleTimeout = 120 * time.Second
	// shutdownTimeout is how long to wait for graceful shutdown.
	shutdownTimeout = 30 * time.Second
)

func main() {
	err := run()
	if err != nil {
		slog.Error("application failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// Create DI container with all services
	c, err := container.New()
	if err != nil {
		return cockroachdberrors.Wrap(err, "failed to create DI container")
	}
	// Get services from container
	cfg := c.Config()
	logger := c.Logger()
	srv := c.Server()
	repo := c.Repository()

	defer func() {
		report := c.Shutdown()
		if !report.Succeed {
			logger.Error("failed to shutdown container", slog.String("error", report.Error()))
		}
	}()

	logger.Info("starting cyberdom site generator",
		slog.Uint64("port", uint64(cfg.Port)),
		slog.String("root_dir", cfg.RootDir),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("cache_enabled", cfg.CacheEnabled),
		slog.Bool("dev_mode", cfg.DevMode),
		slog.Duration("timeout", cfg.Timeout),
	)

	// Print configuration for visibility
	logger.Info("configuration", slog.String("config", cfg.String()))

	logger.Info("content repository initialized",
		slog.Time("last_modified", repo.LastModified()),
	)

	// Setup Gin
	if cfg.LogLevel == "info" || cfg.LogLevel == "warn" || cfg.LogLevel == "error" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(logger))

	// Register server routes
	srv.RegisterRoutes(router)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  idleTimeout,
	}

	// Setup graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine
	errChan := make(chan error, 1)

	go func() {
		logger.Info("server starting", slog.String("address", httpServer.Addr))

		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	// Start file watcher in dev mode
	if cfg.DevMode {
		go watchForChanges(cfg.RootDir, repo, logger)

		logger.Info("file watcher started in dev mode")
	}

	// Wait for shutdown signal or error
	select {
	case err := <-errChan:
		return cockroachdberrors.Wrap(err, "server error")
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return cockroachdberrors.Wrap(err, "server shutdown failed")
	}

	logger.Info("server stopped gracefully")

	return nil
}

// requestLogger is a gin middleware that logs HTTP requests.
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log after request
		duration := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("http request",
			slog.String("client_ip", clientIP),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", statusCode),
			slog.Duration("duration", duration),
			slog.Int("errors", len(c.Errors)),
		)
	}
}

// Compile-time check that Server implements the expected interface.
var _ *server.Server = (*server.Server)(nil)
