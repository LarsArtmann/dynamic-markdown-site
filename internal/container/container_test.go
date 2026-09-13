package container

import (
	"context"
	"os"
	"os/exec"
	"testing"
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

// TestNew tests that a new container can be created.
// Note: This test runs in a subprocess because container.New() ultimately
// calls config.Load() which uses flag.Parse() - and flag.Parse() can only
// be called once per process.
func TestNew(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		// We're in the subprocess - run the actual test
		container, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		if container == nil {
			t.Fatal("New() returned nil container")
		}

		// Clean shutdown
		assertNoShutdownError(t, container)

		return
	}

	runInSubprocess(t)
}

// TestContainerServices tests that all services can be accessed from the container.
func TestContainerServices(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		container, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		defer func() { _ = container.Shutdown() }()

		// Test Config accessor
		cfg, err := container.Config()
		if err != nil {
			t.Errorf("Config() error: %v", err)
		} else if cfg == nil {
			t.Error("Config() returned nil")
		}

		// Test Logger accessor
		logger, err := container.Logger()
		if err != nil {
			t.Errorf("Logger() error: %v", err)
		} else if logger == nil {
			t.Error("Logger() returned nil")
		}

		// Test Repository accessor
		repo, err := container.Repository()
		if err != nil {
			t.Errorf("Repository() error: %v", err)
		} else if repo == nil {
			t.Error("Repository() returned nil")
		}

		// Test Server accessor
		server, err := container.Server()
		if err != nil {
			t.Errorf("Server() error: %v", err)
		} else if server == nil {
			t.Error("Server() returned nil")
		}

		return
	}

	runInSubprocess(t)
}

// TestContainerShutdown tests that shutdown works properly.
func TestContainerShutdown(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		container, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		// Access some services to ensure they're initialized
		if _, err := container.Config(); err != nil {
			t.Errorf("Config() error: %v", err)
		}

		if _, err := container.Logger(); err != nil {
			t.Errorf("Logger() error: %v", err)
		}

		// Shutdown should succeed (do.ShutdownReport returns non-nil but empty Error() on success)
		assertNoShutdownError(t, container)

		// Second shutdown should be safe (idempotent)
		assertNoShutdownError(t, container)

		return
	}

	runInSubprocess(t)
}

// TestContainerMultipleAccess tests that services can be accessed multiple times.
func TestContainerMultipleAccess(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		container, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		defer func() { _ = container.Shutdown() }()

		// Access services multiple times - should return same instances (singleton)
		cfg1, err := container.Config()
		if err != nil {
			t.Fatalf("Config() error: %v", err)
		}

		cfg2, err := container.Config()
		if err != nil {
			t.Fatalf("Config() error: %v", err)
		}

		if cfg1 != cfg2 {
			t.Error("Config() should return same instance (singleton)")
		}

		logger1, err := container.Logger()
		if err != nil {
			t.Fatalf("Logger() error: %v", err)
		}

		logger2, err := container.Logger()
		if err != nil {
			t.Fatalf("Logger() error: %v", err)
		}

		if logger1 != logger2 {
			t.Error("Logger() should return same instance (singleton)")
		}

		repo1, err := container.Repository()
		if err != nil {
			t.Fatalf("Repository() error: %v", err)
		}

		repo2, err := container.Repository()
		if err != nil {
			t.Fatalf("Repository() error: %v", err)
		}

		if repo1 != repo2 {
			t.Error("Repository() should return same instance (singleton)")
		}

		server1, err := container.Server()
		if err != nil {
			t.Fatalf("Server() error: %v", err)
		}

		server2, err := container.Server()
		if err != nil {
			t.Fatalf("Server() error: %v", err)
		}

		if server1 != server2 {
			t.Error("Server() should return same instance (singleton)")
		}

		return
	}

	runInSubprocess(t)
}

// TestContainerServiceOrder tests that services can be accessed in any order.
func TestContainerServiceOrder(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		container, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}

		defer func() { _ = container.Shutdown() }()

		// Access services in different order than registration
		// This verifies that dependency resolution works correctly
		server := container.Server()     // Depends on repo, searcher, logger, cache
		searcher := container.Searcher() // Depends on repo
		renderer := container.Renderer() // No dependencies
		repo := container.Repository()   // Depends on config
		cache := container.Cache()       // No dependencies
		logger := container.Logger()     // Depends on config
		cfg := container.Config()        // No dependencies (loaded from flags/env)

		// All should be non-nil
		if server == nil {
			t.Error("Server() returned nil")
		}

		if searcher == nil {
			t.Error("Searcher() returned nil")
		}

		if renderer == nil {
			t.Error("Renderer() returned nil")
		}

		if repo == nil {
			t.Error("Repository() returned nil")
		}

		if cache == nil {
			t.Error("Cache() returned nil")
		}

		if logger == nil {
			t.Error("Logger() returned nil")
		}

		if cfg == nil {
			t.Error("Config() returned nil")
		}

		return
	}

	// Run test in subprocess
	cmd := exec.CommandContext(
		context.Background(),
		os.Args[0],
		"-test.run=TestContainerServiceOrder",
	)

	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Subprocess failed: %v\nOutput: %s", err, output)
	}
}
