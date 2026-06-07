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
	"github.com/larsartmann/dynamic-markdown-site/internal/config"
	"github.com/larsartmann/dynamic-markdown-site/internal/container"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
)

const (
	idleTimeout     = 120 * time.Second
	shutdownTimeout = 30 * time.Second
)

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

	httpServer := setupHTTPServer(svc)

	startFileWatcher(svc)

	if err := serveHTTP(svc, httpServer); err != nil {
		shutdownServices(svc)

		return err
	}

	return gracefulShutdown(svc, httpServer)
}

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

func logStartupInfo(svc *services) {
	svc.logger.Info(
		"starting site generator",
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

	svc.logger.Info(
		"content repository initialized",
		slog.Time("last_modified", svc.repo.LastModified()),
	)
}

func setupHTTPServer(svc *services) *http.Server {
	handler := svc.server.Handler()

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", svc.config.Port),
		Handler:           handler,
		ReadTimeout:       svc.config.Timeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      svc.config.Timeout,
		IdleTimeout:       idleTimeout,
		ErrorLog:          slog.NewLogLogger(svc.logger.Handler(), slog.LevelError),
	}
}

func serveHTTP(svc *services, httpServer *http.Server) error {
	errChan := make(chan error, 1)

	go func() {
		svc.logger.Info("server starting", slog.String("address", httpServer.Addr))

		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
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

func gracefulShutdown(svc *services, httpServer *http.Server) error {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	err := httpServer.Shutdown(shutdownCtx)
	if err != nil {
		return cockroachdberrors.Wrapf(err, "server shutdown failed (addr: %s)", httpServer.Addr)
	}

	shutdownServices(svc)

	svc.logger.Info("server stopped gracefully")

	return nil
}

func shutdownServices(svc *services) {
	svc.server.Shutdown()

	report := svc.container.Shutdown()
	if !report.Succeed {
		svc.logger.Error("failed to shutdown container", slog.String("error", report.Error()))
	}
}

func startFileWatcher(svc *services) {
	if svc.config.DevMode {
		liveReload := svc.server.LiveReload()
		go watchForChanges(svc.config.RootDir, svc.repo, liveReload, svc.logger)

		svc.logger.Info("file watcher started in dev mode")
	}
}
