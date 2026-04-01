// Package config handles application configuration.
package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

// Sentinel errors for validation failures.
var (
	errInvalidPort     = errors.New("invalid port (must be 1-65535)")
	errInvalidRootDir  = errors.New("root path is not a directory")
	errInvalidLogLevel = errors.New("invalid log level (must be debug, info, warn, or error)")
)

// Config holds all application configuration.
type Config struct {
	Port         uint16
	RootDir      string
	StorageURL   string
	LogLevel     string
	CacheEnabled bool
	DevMode      bool
	Timeout      time.Duration
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		RootDir:      ".",
		StorageURL:   "",
		LogLevel:     "info",
		CacheEnabled: true,
		DevMode:      false,
		Timeout:      30 * time.Second,
	}
}

// Load loads configuration from flags, environment, and defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Define flags
	portFlag := flag.Int("port", int(cfg.Port), "Port to run the server on")
	flag.StringVar(&cfg.RootDir, "root", cfg.RootDir, "Root directory containing markdown files")
	flag.StringVar(&cfg.RootDir, "r", cfg.RootDir, "Root directory (shorthand)")
	flag.StringVar(&cfg.LogLevel, "log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	flag.StringVar(&cfg.StorageURL, "storage-url", cfg.StorageURL, "Blob storage URL (e.g., file:///path, s3://bucket/prefix, gs://bucket/prefix)")
	flag.BoolVar(&cfg.CacheEnabled, "cache", cfg.CacheEnabled, "Enable response caching")
	flag.BoolVar(&cfg.DevMode, "dev", cfg.DevMode, "Development mode (disables caching)")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "Request timeout")

	// Parse flags
	flag.Parse()

	// Apply parsed flags
	if *portFlag > 0 && *portFlag <= 65535 {
		cfg.Port = uint16(*portFlag)
	}

	// Override with environment variables
	if port := os.Getenv("DYNAMIC_MARKDOWN_PORT"); port != "" {
		if p, err := parseUint16(port); err == nil {
			cfg.Port = p
		}
	}

	if root := os.Getenv("DYNAMIC_MARKDOWN_ROOT"); root != "" {
		cfg.RootDir = root
	}

	if level := os.Getenv("DYNAMIC_MARKDOWN_LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	if storageURL := os.Getenv("DYNAMIC_MARKDOWN_STORAGE_URL"); storageURL != "" {
		cfg.StorageURL = storageURL
	}

	if cache := os.Getenv("DYNAMIC_MARKDOWN_CACHE"); cache != "" {
		cfg.CacheEnabled = parseBool(cache)
	}

	if dev := os.Getenv("DYNAMIC_MARKDOWN_DEV"); dev != "" {
		cfg.DevMode = parseBool(dev)
	}

	if timeout := os.Getenv("DYNAMIC_MARKDOWN_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			cfg.Timeout = d
		}
	}

	// Post-process configuration
	err := cfg.validate()
	if err != nil {
		return nil, err
	}

	// Apply derived settings
	if cfg.DevMode {
		cfg.CacheEnabled = false
	}

	// Convert relative root dir to absolute and normalize (only for filesystem)
	if cfg.StorageURL == "" && !filepath.IsAbs(cfg.RootDir) {
		abs, err := filepath.Abs(cfg.RootDir)
		if err != nil {
			return nil, errors.Wrap(err, "resolve root directory")
		}

		cfg.RootDir = abs
	}

	// Normalize path to remove trailing slashes and clean up (only for filesystem)
	if cfg.StorageURL == "" {
		cfg.RootDir = filepath.Clean(cfg.RootDir)
	}

	return cfg, nil
}

// validate checks that configuration is valid.
func (c *Config) validate() error {
	// Validate port
	if c.Port == 0 {
		return errors.Wrap(errInvalidPort, fmt.Sprintf("port=%d", c.Port))
	}

	// Validate storage - either StorageURL (blob) or RootDir (filesystem)
	if c.StorageURL == "" {
		// Using filesystem storage
		info, err := os.Stat(c.RootDir)
		if err != nil {
			return errors.Wrap(err, "root directory does not exist")
		}

		if !info.IsDir() {
			return errors.Wrap(errInvalidRootDir, "path="+c.RootDir)
		}
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(c.LogLevel)] {
		return errors.Wrap(errInvalidLogLevel, "level="+c.LogLevel)
	}

	return nil
}

// LogValue returns a slog.Value for structured logging.
func (c *Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Uint64("port", uint64(c.Port)),
		slog.String("root_dir", c.RootDir),
		slog.String("storage_url", c.StorageURL),
		slog.String("log_level", c.LogLevel),
		slog.Bool("cache_enabled", c.CacheEnabled),
		slog.Bool("dev_mode", c.DevMode),
		slog.Duration("timeout", c.Timeout),
	)
}

// SlogLevel returns the slog.Level corresponding to LogLevel.
func (c *Config) SlogLevel() slog.Level {
	switch strings.ToLower(c.LogLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// String returns a string representation of the config.
func (c *Config) String() string {
	var b strings.Builder
	b.WriteString("Configuration:\n")
	fmt.Fprintf(&b, "  Port:         %d\n", c.Port)
	if c.StorageURL != "" {
		fmt.Fprintf(&b, "  Storage URL:  %s\n", c.StorageURL)
	} else {
		fmt.Fprintf(&b, "  Root Dir:     %s\n", c.RootDir)
	}
	fmt.Fprintf(&b, "  Log Level:    %s\n", c.LogLevel)
	fmt.Fprintf(&b, "  Cache:        %v\n", c.CacheEnabled)
	fmt.Fprintf(&b, "  Dev Mode:     %v\n", c.DevMode)
	fmt.Fprintf(&b, "  Timeout:      %v\n", c.Timeout)

	return b.String()
}

// parseBool parses a boolean string.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// parseUint16 parses a uint16 string.
func parseUint16(s string) (uint16, error) {
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, errors.Wrap(err, "parse port number")
	}

	return uint16(val), nil
}
