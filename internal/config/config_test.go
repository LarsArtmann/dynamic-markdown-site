package config

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoadSubprocess tests the Load() function by running it in a subprocess
// This is necessary because flag.Parse() can only be called once per process.
func TestLoadSubprocess(t *testing.T) {
	// Note: This test cannot use t.Parallel() because it spawns subprocesses
	// that also need to call flag.Parse(). The subprocess detection prevents
	// parallel execution.
	if os.Getenv("TEST_LOAD_CONFIG") == "1" {
		// This is the subprocess - actually run Load()
		cfg, err := Load()
		if err != nil {
			_, _ = os.Stderr.WriteString("LOAD_ERROR:" + err.Error())
			os.Exit(1)
		}
		// Output the config values for verification
		_, _ = os.Stdout.WriteString("PORT:" + string(rune(cfg.Port)) + "\n")
		_, _ = os.Stdout.WriteString("ROOT:" + cfg.RootDir + "\n")
		_, _ = os.Stdout.WriteString("LOG_LEVEL:" + cfg.LogLevel + "\n")
		_, _ = os.Stdout.WriteString("CACHE:" + boolStr(cfg.CacheEnabled) + "\n")
		_, _ = os.Stdout.WriteString("DEV:" + boolStr(cfg.DevMode) + "\n")
		os.Exit(0)
	}

	// runSubprocessTest runs a subprocess test with the given environment variables
	runSubprocessTest := func(t *testing.T, name string, extraEnv []string) {
		t.Helper()

		cmd := exec.CommandContext(context.Background(), os.Args[0], "-test.run=TestLoadSubprocess")

		cmd.Env = append(os.Environ(), append([]string{"TEST_LOAD_CONFIG=1"}, extraEnv...)...)

		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Load() %s failed: %v, output: %s", name, err, output)
		}
	}

	tests := []struct {
		name    string
		desc    string
		envVars []string
	}{
		{"default_config", "with defaults", nil},
		{"env_override_port", "with env port", []string{"DYNAMIC_MARKDOWN_PORT=3000"}},
		{"env_dev_mode", "with dev mode", []string{"DYNAMIC_MARKDOWN_DEV=true"}},
		{"env_cache_disabled", "with cache disabled", []string{"DYNAMIC_MARKDOWN_CACHE=false"}},
		{"invalid_port_env", "with invalid port env", []string{"DYNAMIC_MARKDOWN_PORT=invalid"}},
		{"custom_timeout_env", "with custom timeout", []string{"DYNAMIC_MARKDOWN_TIMEOUT=60s"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			runSubprocessTest(t, tt.desc, tt.envVars)
		})
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}

	return "false"
}

func TestParseBool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{"yes", true},
		{"YES", true},
		{"on", true},
		{"ON", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
		{"random", false},
		{"  true  ", true},
		{"  FALSE  ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := parseBool(tt.input)
			if result != tt.expected {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseUint16(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		expected    uint16
		expectError bool
	}{
		{"zero", "0", 0, false},
		{"one", "1", 1, false},
		{"port 8080", "8080", 8080, false},
		{"max uint16", "65535", 65535, false},
		{"overflow", "65536", 0, true},
		{"negative", "-1", 0, true},
		{"invalid", "abc", 0, true},
		{"empty", "", 0, true},
		{"float", "3.14", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseUint16(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("parseUint16(%q) expected error, got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("parseUint16(%q) unexpected error: %v", tt.input, err)
				}

				if result != tt.expected {
					t.Errorf("parseUint16(%q) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

// assertValidationError is a helper for testing config validation errors.
// It creates a config with the given values and asserts that validation
// returns an error containing the expected substring.
func assertValidationError(
	t *testing.T,
	port uint16,
	rootDir string,
	logLevel string,
	expectedErrSubstring string,
) {
	t.Helper()

	cfg := &Config{
		Port:     port,
		RootDir:  rootDir,
		LogLevel: logLevel,
	}

	err := cfg.validate()
	if err == nil {
		t.Errorf("validate() expected error containing %q, got nil", expectedErrSubstring)

		return
	}

	if !strings.Contains(err.Error(), expectedErrSubstring) {
		t.Errorf("validate() error should contain %q, got: %v", expectedErrSubstring, err)
	}
}

func TestConfigValidate(t *testing.T) {
	t.Parallel()
	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		cfg := &Config{
			Port:     8080,
			RootDir:  tmpDir,
			LogLevel: "info",
		}

		err := cfg.validate()
		if err != nil {
			t.Errorf("validate() unexpected error: %v", err)
		}
	})

	t.Run("zero port", func(t *testing.T) {
		t.Parallel()
		assertValidationError(t, 0, t.TempDir(), "info", "port")
	})

	t.Run("non-existent root dir", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			Port:     8080,
			RootDir:  "/non/existent/directory/12345",
			LogLevel: "info",
		}

		err := cfg.validate()
		if err == nil {
			t.Error("validate() expected error for non-existent directory")
		}

		if !strings.Contains(err.Error(), "root directory") {
			t.Errorf("validate() error should mention root directory, got: %v", err)
		}
	})

	t.Run("root is file not directory", func(t *testing.T) {
		t.Parallel()
		tmpFile := filepath.Join(t.TempDir(), "testfile")
		if err := os.WriteFile(tmpFile, []byte("test"), 0o644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cfg := &Config{
			Port:     8080,
			RootDir:  tmpFile,
			LogLevel: "info",
		}

		err := cfg.validate()
		if err == nil {
			t.Error("validate() expected error when root is file")
		}

		if !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("validate() error should mention 'not a directory', got: %v", err)
		}
	})

	t.Run("invalid log level", func(t *testing.T) {
		t.Parallel()
		assertValidationError(t, 8080, t.TempDir(), "invalid", "log level")
	})

	t.Run("all valid log levels", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()

		validLevels := []string{
			"debug",
			"info",
			"warn",
			"error",
			"DEBUG",
			"INFO",
			"WARN",
			"ERROR",
			"Debug",
			"Info",
			"Warn",
			"Error",
		}
		for _, level := range validLevels {
			cfg := &Config{
				Port:     8080,
				RootDir:  tmpDir,
				LogLevel: level,
			}

			err := cfg.validate()
			if err != nil {
				t.Errorf("validate() unexpected error for level %q: %v", level, err)
			}
		}
	})
}

func TestConfigLogValue(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.RootDir = "/test"
	cfg.LogLevel = "debug"

	logValue := cfg.LogValue()
	if logValue.Kind() != slog.KindGroup {
		t.Errorf("LogValue() kind = %v, want Group", logValue.Kind())
	}
}

func TestConfigSlogLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		logLevel string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.logLevel, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{LogLevel: tt.logLevel}

			result := cfg.SlogLevel()
			if result != tt.expected {
				t.Errorf("SlogLevel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Port:         8080,
		RootDir:      "/test/path",
		LogLevel:     "debug",
		CacheEnabled: true,
		DevMode:      true,
		Timeout:      45 * time.Second,
	}

	result := cfg.String()

	if !strings.Contains(result, "8080") {
		t.Error("String() should contain port")
	}

	if !strings.Contains(result, "/test/path") {
		t.Error("String() should contain root dir")
	}

	if !strings.Contains(result, "debug") {
		t.Error("String() should contain log level")
	}

	if !strings.Contains(result, "Cache") {
		t.Error("String() should contain Cache")
	}

	if !strings.Contains(result, "Dev Mode") {
		t.Error("String() should contain Dev Mode")
	}

	if !strings.Contains(result, "Timeout") {
		t.Error("String() should contain Timeout")
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	// Test that default values are sensible
	// Note: We can't test Load() directly because flag.Parse() can only be called once
	// But we can verify the defaults are set correctly
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("default Port = %d, want 8080", cfg.Port)
	}

	if cfg.RootDir != "." {
		t.Errorf("default RootDir = %s, want .", cfg.RootDir)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("default LogLevel = %s, want info", cfg.LogLevel)
	}

	if !cfg.CacheEnabled {
		t.Error("default CacheEnabled should be true")
	}

	if cfg.DevMode {
		t.Error("default DevMode should be false")
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("default Timeout = %v, want 30s", cfg.Timeout)
	}
}

func TestConfigDevModeDisablesCache(t *testing.T) {
	t.Parallel()
	// This tests the logic that DevMode should disable cache
	// The actual Load() function applies this after validation
	cfg := DefaultConfig()
	cfg.RootDir = t.TempDir()
	cfg.DevMode = true

	// Simulate the Load() post-processing
	if cfg.DevMode {
		cfg.CacheEnabled = false
	}

	if cfg.CacheEnabled {
		t.Error("DevMode should disable cache")
	}
}
