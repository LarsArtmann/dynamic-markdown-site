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
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
	"github.com/larsartmann/go-health"
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
	_ do.HealthcheckerWithContext = (*content.FileSystemRepository)(nil)
	_ do.HealthcheckerWithContext = (*content.BlobRepository)(nil)
	_ do.HealthcheckerWithContext = (*content.InMemoryRepository)(nil)
	_ do.HealthcheckerWithContext = (*renderer.GoldmarkRenderer)(nil)
)

// probeRefreshInterval bounds how often the go-health probe re-runs the DI
// health sweep. The readiness handler serves the cached verdict between runs,
// so orchestrator probes never fan out into the repository.
const probeRefreshInterval = 15 * time.Second

// Container holds the DI injector and provides access to services.
type Container struct {
	injector do.Injector
	probe    *health.Probe
}

// New creates a new DI container with all services registered.
func New() (*Container, error) {
	injector := do.New()

	// Register providers - order doesn't matter, dependencies are resolved automatically
	healthMetrics := server.NewHealthMetrics()
	do.ProvideValue(injector, healthMetrics)

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

	probe := health.New(injector,
		// The repository is the only dependency whose failure makes the site
		// unusable: it alone flips readiness to 503. The renderer's missing-D2
		// degradation stays non-critical (warn, HTTP 200).
		health.WithCriticalService[content.Repository](),
		health.WithVersion(version.Version),
		health.WithRefreshInterval(probeRefreshInterval),
		health.WithEvaluationHook(healthMetrics.Observe),
	)

	// Start runs the first sweep synchronously and fails fast when a critical
	// service name never produced a check (typo guard, ErrUnknownCriticalService).
	if err := probe.Start(context.Background()); err != nil {
		_ = injector.Shutdown()

		return nil, errors.Wrap(err, "failed to start health probe")
	}

	return &Container{injector: injector, probe: probe}, nil
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

// Probe returns the go-health probe wired over the injector. Nil for
// containers built directly in tests without a probe.
func (c *Container) Probe() *health.Probe {
	return c.probe
}

// HealthCheck probes every invoked service through samber/do's sweep. The
// returned map holds one entry per service: non-nil for a failed health
// check, nil for a passing check or a service without a Healthchecker
// interface (so "any non-nil value means unhealthy", while "all nil means
// healthy").
func (c *Container) HealthCheck(ctx context.Context) map[string]error {
	return c.injector.HealthCheckWithContext(ctx)
}

// Shutdown gracefully shuts down all services with a background context. The
// probe stops first so its refresh loop cannot sweep services mid-teardown;
// the injector then runs its dependency-aware shutdown wave.
func (c *Container) Shutdown() *do.ShutdownReport {
	return c.ShutdownWithContext(context.Background())
}

// ShutdownWithContext is Shutdown with a deadline: the injector runs its
// shutdown wave under ctx, and the report records whichever services did not
// finish in time.
func (c *Container) ShutdownWithContext(ctx context.Context) *do.ShutdownReport {
	if c.probe != nil {
		c.probe.Shutdown()
	}

	return c.injector.ShutdownWithContext(ctx)
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

	renderer, err := do.Invoke[*renderer.GoldmarkRenderer](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve renderer")
	}

	cfg, err := do.Invoke[*config.Config](i)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve config")
	}

	// Optional service: containers built without health metrics still serve,
	// with /metrics omitting the health section.
	healthMetrics, _ := do.Invoke[*server.HealthMetrics](i)

	return server.NewServer(repo, searcher, logger, htmlCache, renderer, healthMetrics, cfg.DevMode, cfg.SiteName), nil
}
