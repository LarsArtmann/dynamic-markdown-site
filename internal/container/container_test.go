package container

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/config"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/dynamic-markdown-site/internal/server"
	"github.com/samber/do/v2"
)

// runInSubprocess runs the current test in a subprocess to isolate flag.Parse() calls.
// This is necessary because container.New() ultimately calls config.Load() which uses
// flag.Parse() - and flag.Parse() can only be called once per process.
func runInSubprocess(t *testing.T) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run="+t.Name())

	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Subprocess failed: %v\nOutput: %s", err, output)
	}
}

// assertNoShutdownError checks that shutdown succeeds or has empty error.
func assertNoShutdownError(t *testing.T, c *Container) {
	t.Helper()

	err := c.Shutdown()
	if err != nil && err.Error() != "" {
		t.Errorf("Shutdown() error: %v", err)
	}
}

func newInProcessContainer(t *testing.T) *Container {
	t.Helper()

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	htmlCache := cache.NewHTMLCache(10)
	t.Cleanup(htmlCache.Close)

	repo := content.NewInMemoryRepository()
	srv := server.NewServer(
		repo,
		content.NewSearcher(repo),
		slog.New(slog.DiscardHandler),
		htmlCache,
		renderer.NewGoldmarkRenderer(),
		false,
		"Test",
	)
	t.Cleanup(srv.Shutdown)

	do.ProvideValue(injector, &config.Config{CacheSize: 10_000})
	do.ProvideValue(injector, slog.New(slog.DiscardHandler))
	do.ProvideValue[content.Repository](injector, repo)
	do.ProvideValue(injector, srv)

	return &Container{injector: injector}
}

func TestAccessors_ReturnInjectedValues(t *testing.T) {
	t.Parallel()

	c := newInProcessContainer(t)

	cfg, err := c.Config()
	if err != nil {
		t.Errorf("Config() error: %v", err)
	} else if cfg == nil {
		t.Error("Config() returned nil")
	}

	logger, err := c.Logger()
	if err != nil {
		t.Errorf("Logger() error: %v", err)
	} else if logger == nil {
		t.Error("Logger() returned nil")
	}

	repo, err := c.Repository()
	if err != nil {
		t.Errorf("Repository() error: %v", err)
	} else if repo == nil {
		t.Error("Repository() returned nil")
	}

	srv, err := c.Server()
	if err != nil {
		t.Errorf("Server() error: %v", err)
	} else if srv == nil {
		t.Error("Server() returned nil")
	}
}

func TestAccessors_PropagateResolveErrors(t *testing.T) {
	t.Parallel()

	c := &Container{injector: do.New()}

	if _, err := c.Config(); err == nil {
		t.Error("Config() should fail when the service is not registered")
	}

	if _, err := c.Logger(); err == nil {
		t.Error("Logger() should fail when the service is not registered")
	}

	if _, err := c.Repository(); err == nil {
		t.Error("Repository() should fail when the service is not registered")
	}

	if _, err := c.Server(); err == nil {
		t.Error("Server() should fail when the service is not registered")
	}
}

func TestShutdown_ReturnsReportInProcess(t *testing.T) {
	t.Parallel()

	c := newInProcessContainer(t)

	report := c.Shutdown()
	if report == nil {
		t.Fatal("Shutdown() returned nil report")
	}

	if errStr := report.Error(); errStr != "" {
		t.Errorf("Shutdown() report error: %v", errStr)
	}
}

func TestProviders_WireFullGraph(t *testing.T) {
	t.Parallel()

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	do.ProvideValue(injector, &config.Config{
		LogLevel:  "debug",
		DevMode:   true,
		RootDir:   t.TempDir(),
		CacheSize: 10_000,
	})
	do.Provide(injector, provideLogger)
	do.Provide(injector, provideCache)
	do.Provide(injector, provideRenderer)
	do.Provide(injector, provideRepository)
	do.Provide(injector, provideSearcher)

	srv, err := provideServer(injector)
	if err != nil {
		t.Fatalf("provideServer() error: %v", err)
	}

	t.Cleanup(srv.Shutdown)

	if srv == nil {
		t.Fatal("provideServer() returned nil server")
	}

	logger, err := do.Invoke[*slog.Logger](injector)
	if err != nil || logger == nil {
		t.Errorf("logger resolution failed: %v", err)
	}

	htmlCache, err := do.Invoke[*cache.HTMLCache](injector)
	if err != nil || htmlCache == nil {
		t.Errorf("cache resolution failed: %v", err)
	}

	searcher, err := do.Invoke[*content.Searcher](injector)
	if err != nil || searcher == nil {
		t.Errorf("searcher resolution failed: %v", err)
	}

	repo, err := do.Invoke[content.Repository](injector)
	if err != nil || repo == nil {
		t.Errorf("repository resolution failed: %v", err)
	}
}

func TestProviders_RepositoryBlobError(t *testing.T) {
	t.Parallel()

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	do.ProvideValue(injector, &config.Config{StorageURL: "unsupported-scheme://nope"})
	do.Provide(injector, provideRepository)

	if _, err := do.Invoke[content.Repository](injector); err == nil {
		t.Error("provideRepository should fail for an unsupported storage URL")
	}
}

func TestProviders_RepositoryMissingConfig(t *testing.T) {
	t.Parallel()

	injector := do.New()
	t.Cleanup(func() { _ = injector.Shutdown() })

	do.Provide(injector, provideRepository)

	if _, err := do.Invoke[content.Repository](injector); err == nil {
		t.Error("provideRepository should fail when config cannot resolve")
	}
}

// TestContainerLifecycle covers the full container lifecycle in one subprocess:
// creation, service access in multiple orders, singleton identity, shutdown,
// and shutdown idempotency. Merging the former five subprocess tests keeps
// every assertion while paying the flag.Parse-isolation subprocess cost once.
func TestContainerLifecycle(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") != "1" {
		runInSubprocess(t)

		return
	}

	container, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if container == nil {
		t.Fatal("New() returned nil container")
	}

	cfg, err := container.Config()
	if err != nil {
		t.Errorf("Config() error: %v", err)
	} else if cfg == nil {
		t.Error("Config() returned nil")
	}

	logger, err := container.Logger()
	if err != nil {
		t.Errorf("Logger() error: %v", err)
	} else if logger == nil {
		t.Error("Logger() returned nil")
	}

	repo, err := container.Repository()
	if err != nil {
		t.Errorf("Repository() error: %v", err)
	} else if repo == nil {
		t.Error("Repository() returned nil")
	}

	srv, err := container.Server()
	if err != nil {
		t.Errorf("Server() error: %v", err)
	} else if srv == nil {
		t.Error("Server() returned nil")
	}

	cfg2, err := container.Config()
	if err != nil {
		t.Fatalf("Config() second access error: %v", err)
	}

	if cfg != cfg2 {
		t.Error("Config() should return same instance (singleton)")
	}

	logger2, err := container.Logger()
	if err != nil {
		t.Fatalf("Logger() second access error: %v", err)
	}

	if logger != logger2 {
		t.Error("Logger() should return same instance (singleton)")
	}

	repo2, err := container.Repository()
	if err != nil {
		t.Fatalf("Repository() second access error: %v", err)
	}

	if repo != repo2 {
		t.Error("Repository() should return same instance (singleton)")
	}

	srv2, err := container.Server()
	if err != nil {
		t.Fatalf("Server() second access error: %v", err)
	}

	if srv != srv2 {
		t.Error("Server() should return same instance (singleton)")
	}

	assertNoShutdownError(t, container)
	assertNoShutdownError(t, container)
}
