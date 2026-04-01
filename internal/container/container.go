// Package container provides dependency injection using samber/do/v2.
package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"charm.land/log/v2"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/config"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/samber/do/v2"
)

// Container holds the DI injector and provides access to services.
type Container struct {
	injector do.Injector
}

// New creates a new DI container with all services registered.
func New() (*Container, error) {
	injector := do.New()

	// Register providers - order doesn't matter, dependencies are resolved automatically
	do.Provide(injector, provideConfig)
	do.Provide(injector, provideLogger)
	do.Provide(injector, provideCache)
	do.Provide(injector, provideRenderer)
	do.Provide(injector, provideRepository)
	do.Provide(injector, provideSearcher)
	do.Provide(injector, provideServer)

	return &Container{injector: injector}, nil
}

// Config returns the application configuration.
func (c *Container) Config() *config.Config {
	return do.MustInvoke[*config.Config](c.injector)
}

// Logger returns the application logger.
func (c *Container) Logger() *slog.Logger {
	return do.MustInvoke[*slog.Logger](c.injector)
}

// Cache returns the HTML cache.
func (c *Container) Cache() *cache.HTMLCache {
	return do.MustInvoke[*cache.HTMLCache](c.injector)
}

// Repository returns the content repository.
func (c *Container) Repository() content.Repository {
	return do.MustInvoke[content.Repository](c.injector)
}

// Renderer returns the markdown renderer.
func (c *Container) Renderer() *renderer.GoldmarkRenderer {
	return do.MustInvoke[*renderer.GoldmarkRenderer](c.injector)
}

// Searcher returns the content searcher.
func (c *Container) Searcher() *content.Searcher {
	return do.MustInvoke[*content.Searcher](c.injector)
}

// Server returns the HTTP server.
func (c *Container) Server() *server.Server {
	return do.MustInvoke[*server.Server](c.injector)
}

// Shutdown gracefully shuts down all services.
func (c *Container) Shutdown() *do.ShutdownReport {
	return c.injector.Shutdown()
}

// Provider functions

func provideConfig(_ do.Injector) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return cfg, nil
}

func provideLogger(i do.Injector) (*slog.Logger, error) {
	cfg := do.MustInvoke[*config.Config](i)

	// Create charmbracelet logger
	logger := log.New(os.Stdout)
	logger.SetReportCaller(cfg.LogLevel == "debug")

	// Set log level from config
	switch cfg.LogLevel {
	case "debug":
		logger.SetLevel(log.DebugLevel)
	case "warn":
		logger.SetLevel(log.WarnLevel)
	case "error":
		logger.SetLevel(log.ErrorLevel)
	default:
		logger.SetLevel(log.InfoLevel)
	}

	// Use JSON format in production, pretty format in dev
	if cfg.DevMode {
		logger.SetFormatter(log.TextFormatter)
	} else {
		logger.SetFormatter(log.JSONFormatter)
	}

	// charmbracelet/log.Logger implements slog.Handler
	return slog.New(logger), nil
}

func provideCache(_ do.Injector) (*cache.HTMLCache, error) {
	// 10,000 entry cache with 1-hour TTL
	return cache.NewHTMLCache(10_000), nil
}

func provideRenderer(_ do.Injector) (*renderer.GoldmarkRenderer, error) {
	// Create diagram renderer for D2 support
	diagramRenderer, err := renderer.NewDiagramRenderer()
	if err != nil {
		// Continue without diagram support if D2 renderer fails
		// We intentionally swallow the error here to degrade gracefully
		// when diagram dependencies are unavailable
		return renderer.NewGoldmarkRenderer(), nil //nolint:nilerr // graceful degradation
	}

	return renderer.NewGoldmarkRendererWithDiagrams(diagramRenderer), nil
}

func provideRepository(i do.Injector) (content.Repository, error) {
	cfg := do.MustInvoke[*config.Config](i)

	// Use blob storage if StorageURL is configured
	if cfg.StorageURL != "" {
		repo, err := content.NewBlobRepository(context.Background(), cfg.StorageURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create blob repository: %w", err)
		}
		return repo, nil
	}

	// Use filesystem repository as default
	repo, err := content.NewFileSystemRepository(cfg.RootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem repository: %w", err)
	}
	return repo, nil
}

func provideSearcher(i do.Injector) (*content.Searcher, error) {
	repo := do.MustInvoke[content.Repository](i)

	return content.NewSearcher(repo), nil
}

func provideServer(i do.Injector) (*server.Server, error) {
	repo := do.MustInvoke[content.Repository](i)
	searcher := do.MustInvoke[*content.Searcher](i)
	logger := do.MustInvoke[*slog.Logger](i)
	cache := do.MustInvoke[*cache.HTMLCache](i)
	cfg := do.MustInvoke[*config.Config](i)

	return server.NewServer(repo, searcher, logger, cache, cfg.DevMode), nil
}
