// Package container provides dependency injection using samber/do/v2.
package container

import (
	"context"
	"log/slog"
	"os"
	"time"

	"charm.land/log/v2"
	"github.com/cockroachdb/errors"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/config"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/samber/do/v2"
)

var errBlobTimeout = errors.New("blob repository creation timed out after 10 seconds")

// Compile-time lifecycle contracts: samber/do dispatches shutdown and health
// sweeps only through these interfaces, so a service losing its lifecycle
// method fails the build here in the composition root instead of leaking
// resources or reporting green at runtime.
var (
	_ do.ShutdownerWithError      = (*content.BlobRepository)(nil)
	_ do.Shutdowner               = (*cache.HTMLCache)(nil)
	_ do.HealthcheckerWithContext = (*server.Server)(nil)
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

	// The server is boot-critical: main starts it immediately after wiring.
	// Resolving it here materializes the provider eagerly, keeping the graph's
	// dependency edges intact (dependency-aware shutdown wave, health sweep
	// sees a constructed service from boot) while construction still fails fast.
	if _, err := do.Invoke[*server.Server](injector); err != nil {
		_ = injector.Shutdown()

		return nil, errors.Wrap(err, "failed to construct server")
	}

	return &Container{injector: injector}, nil
}

// Config returns the application configuration.
func (c *Container) Config() (*config.Config, error) {
	return do.Invoke[*config.Config](c.injector)
}

// Logger returns the application logger.
func (c *Container) Logger() (*slog.Logger, error) {
	return do.Invoke[*slog.Logger](c.injector)
}

// Repository returns the content repository.
func (c *Container) Repository() (content.Repository, error) {
	return do.Invoke[content.Repository](c.injector)
}

// Server returns the HTTP server.
func (c *Container) Server() (*server.Server, error) {
	return do.Invoke[*server.Server](c.injector)
}

// HealthCheck probes every service implementing samber/do's Healthchecker
// interfaces (the server today). The returned map is keyed by service
// description; an empty map means every probed service is healthy.
func (c *Container) HealthCheck(ctx context.Context) map[string]error {
	return c.injector.HealthCheckWithContext(ctx)
}

// Shutdown gracefully shuts down all services.
func (c *Container) Shutdown() *do.ShutdownReport {
	return c.injector.Shutdown()
}

// Provider functions

func provideConfig(_ do.Injector) (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	return cfg, nil
}

func provideLogger(i do.Injector) (*slog.Logger, error) {
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve config")
	}

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

func provideCache(i do.Injector) (*cache.HTMLCache, error) {
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve config")
	}

	return cache.NewHTMLCache(cfg.CacheSize), nil
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
	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve config")
	}

	// Use blob storage if StorageURL is configured
	if cfg.StorageURL != "" {
		// Use a channel-based timeout since gocloud.dev's blob.OpenBucket doesn't
		// respect context cancellation for GCS credential discovery
		type result struct {
			repo content.Repository
			err  error
		}

		resultCh := make(chan result, 1)

		go func() {
			repo, err := content.NewBlobRepository(context.Background(), cfg.StorageURL)
			resultCh <- result{repo: repo, err: err}
		}()

		select {
		case res := <-resultCh:
			if res.err != nil {
				return nil, errors.Wrap(res.err, "failed to create blob repository")
			}

			return res.repo, nil
		case <-time.After(10 * time.Second):
			return nil, errors.Wrap(errBlobTimeout, "storage_url="+cfg.StorageURL)
		}
	}

	// Use filesystem repository as default
	repo, err := content.NewFileSystemRepository(cfg.RootDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create filesystem repository")
	}

	return repo, nil
}

func provideSearcher(i do.Injector) (*content.Searcher, error) {
	repo, err := do.Invoke[content.Repository](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve repository")
	}

	return content.NewSearcher(repo), nil
}

func provideServer(i do.Injector) (*server.Server, error) {
	repo, err := do.Invoke[content.Repository](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve repository")
	}

	searcher, err := do.Invoke[*content.Searcher](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve searcher")
	}

	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve logger")
	}

	htmlCache, err := do.Invoke[*cache.HTMLCache](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve cache")
	}

	rndr, err := do.Invoke[*renderer.GoldmarkRenderer](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve renderer")
	}

	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve config")
	}

	return server.NewServer(repo, searcher, logger, htmlCache, rndr, cfg.DevMode, cfg.SiteName), nil
}
