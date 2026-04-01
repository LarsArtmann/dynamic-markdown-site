// Dynamic Markdown Site - Main entry point
//
// A type-safe, high-performance markdown-to-website converter.
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
	"github.com/larsartmann/dynamic-markdown-site/internal/config"
	"github.com/larsartmann/dynamic-markdown-site/internal/container"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
)

// Server timeouts.
const (
	// idleTimeout is how long to wait for idle connections before closing.
	idleTimeout = 120 * time.Second
	// shutdownTimeout is how long to wait for graceful shutdown.
	shutdownTimeout = 30 * time.Second
)

// services holds all services obtained from the container.
type services struct {
	config    *config.Config
	logger    *slog.Logger
	server    *server.Server
	repo      content.Repository
	container *container.Container
}

func main() {
	if version.Version == "dev" {
		slog.Info("running in development mode (version not set)")
	}
	err := run()
	if err != nil {
		slog.Error("application failed", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	svc, err := setupServices()
	if err != nil {
		return err
	}

	router, httpServer := setupServer(svc)

	startFileWatcher(svc)

	if err := serveHTTP(svc, httpServer, router); err != nil {
		shutdownServices(svc)
		return err
	}

	return gracefulShutdown(svc, httpServer)
}

// setupServices creates the DI container and extracts all services.
func setupServices() (*services, error) {
	c, err := container.New()
	if err != nil {
		return nil, cockroachdberrors.Wrap(err, "failed to create DI container")
	}

	svc := &services{
		config:    c.Config(),
		logger:    c.Logger(),
		server:    c.Server(),
		repo:      c.Repository(),
		container: c,
	}

	logStartupInfo(svc)

	return svc, nil
}

// logStartupInfo logs all startup configuration.
func logStartupInfo(svc *services) {
	svc.logger.Info("starting site generator",
		slog.Uint64("port", uint64(svc.config.Port)),
		slog.String("root_dir", svc.config.RootDir),
		slog.String("log_level", svc.config.LogLevel),
		slog.Bool("cache_enabled", svc.config.CacheEnabled),
		slog.Bool("dev_mode", svc.config.DevMode),
		slog.Duration("timeout", svc.config.Timeout),
		slog.String("version", version.Version),
		slog.String("commit", version.Commit),
		slog.String("build_date", version.BuildDate),
	)

	svc.logger.Info("configuration", slog.String("config", svc.config.String()))

	svc.logger.Info("content repository initialized",
		slog.Time("last_modified", svc.repo.LastModified()),
	)
}

// setupServer creates the router and HTTP server.
func setupServer(svc *services) (*gin.Engine, *http.Server) {
	configureGin(svc.config.LogLevel)

	router := createRouter(svc)

	//nolint:exhaustruct
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", svc.config.Port),
		Handler:      router,
		ReadTimeout:  svc.config.Timeout,
		WriteTimeout: svc.config.Timeout,
		IdleTimeout:  idleTimeout,
		ErrorLog:     slog.NewLogLogger(svc.logger.Handler(), slog.LevelError),
	}

	return router,
		httpServer
}

// configureGin sets the Gin mode based on log level.
func configureGin(logLevel string) {
	if logLevel == "info" || logLevel == "warn" || logLevel == "error" {
		gin.SetMode(gin.ReleaseMode)
	}
}

// createRouter creates and configures the Gin router.
func createRouter(svc *services) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger(svc.logger))

	svc.server.RegisterRoutes(router)

	return router
}

// serveHTTP starts the HTTP server and waits for shutdown.
func serveHTTP(svc *services, httpServer *http.Server, _ *gin.Engine) error {
	errChan := make(chan error, 1)

	go func() {
		svc.logger.Info("server starting", slog.String("address", httpServer.Addr))

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errChan:
		return cockroachdberrors.Wrapf(err, "server error (addr: %s)", httpServer.Addr)
	case <-ctx.Done():
		svc.logger.Info("shutdown signal received")
		return nil
	}
}

// gracefulShutdown performs the HTTP server shutdown.
func gracefulShutdown(svc *services, httpServer *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return cockroachdberrors.Wrapf(err, "server shutdown failed (addr: %s)", httpServer.Addr)
	}

	shutdownServices(svc)

	svc.logger.Info("server stopped gracefully")

	return nil
}

// shutdownServices performs graceful shutdown of services.
func shutdownServices(svc *services) {
	report := svc.container.Shutdown()
	if !report.Succeed {
		svc.logger.Error("failed to shutdown container", slog.String("error", report.Error()))
	}
}

// startFileWatcher starts the file watcher in dev mode.
func startFileWatcher(svc *services) {
	if svc.config.DevMode {
		liveReload := svc.server.LiveReload()
		go watchForChanges(svc.config.RootDir, svc.repo, liveReload, svc.logger)
		svc.logger.Info("file watcher started in dev mode")
	}
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
