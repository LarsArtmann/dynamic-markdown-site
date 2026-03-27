package container

import (
	"os"
	"os/exec"
	"testing"
)

// runInSubprocess runs the current test in a subprocess to isolate flag.Parse() calls.
// This is necessary because container.New() ultimately calls config.Load() which uses
// flag.Parse() - and flag.Parse() can only be called once per process.
func runInSubprocess(t *testing.T) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run="+t.Name())

	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Subprocess failed: %v\nOutput: %s", err, output)
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
		if err := container.Shutdown(); err != nil && err.Error() != "" {
			t.Errorf("Shutdown() error: %v", err)
		}

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
		cfg := container.Config()
		if cfg == nil {
			t.Error("Config() returned nil")
		}

		// Test Logger accessor
		logger := container.Logger()
		if logger == nil {
			t.Error("Logger() returned nil")
		}

		// Test Cache accessor
		cache := container.Cache()
		if cache == nil {
			t.Error("Cache() returned nil")
		}

		// Test Repository accessor
		repo := container.Repository()
		if repo == nil {
			t.Error("Repository() returned nil")
		}

		// Test Renderer accessor
		renderer := container.Renderer()
		if renderer == nil {
			t.Error("Renderer() returned nil")
		}

		// Test Searcher accessor
		searcher := container.Searcher()
		if searcher == nil {
			t.Error("Searcher() returned nil")
		}

		// Test Server accessor
		server := container.Server()
		if server == nil {
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
		_ = container.Config()
		_ = container.Logger()
		_ = container.Cache()

		// Shutdown should succeed (do.ShutdownReport returns non-nil but empty Error() on success)
		if err := container.Shutdown(); err != nil && err.Error() != "" {
			t.Errorf("Shutdown() error: %v", err)
		}

		// Second shutdown should be safe (idempotent)
		if err := container.Shutdown(); err != nil && err.Error() != "" {
			t.Errorf("Second Shutdown() error: %v", err)
		}

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
		cfg1 := container.Config()

		cfg2 := container.Config()
		if cfg1 != cfg2 {
			t.Error("Config() should return same instance (singleton)")
		}

		logger1 := container.Logger()

		logger2 := container.Logger()
		if logger1 != logger2 {
			t.Error("Logger() should return same instance (singleton)")
		}

		cache1 := container.Cache()

		cache2 := container.Cache()
		if cache1 != cache2 {
			t.Error("Cache() should return same instance (singleton)")
		}

		renderer1 := container.Renderer()

		renderer2 := container.Renderer()
		if renderer1 != renderer2 {
			t.Error("Renderer() should return same instance (singleton)")
		}

		repo1 := container.Repository()

		repo2 := container.Repository()
		if repo1 != repo2 {
			t.Error("Repository() should return same instance (singleton)")
		}

		searcher1 := container.Searcher()

		searcher2 := container.Searcher()
		if searcher1 != searcher2 {
			t.Error("Searcher() should return same instance (singleton)")
		}

		server1 := container.Server()

		server2 := container.Server()
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
	cmd := exec.Command(os.Args[0], "-test.run=TestContainerServiceOrder")

	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Subprocess failed: %v\nOutput: %s", err, output)
	}
}
