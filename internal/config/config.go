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
	SiteName     string
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
		SiteName:     "Site",
	}
}

// Load loads configuration from flags, environment, and defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	cfg.defineAndParseFlags()
	cfg.applyEnvironmentOverrides()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if err := cfg.applyDerivedSettings(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// defineAndParseFlags registers and parses command-line flags.
func (c *Config) defineAndParseFlags() {
	portFlag := flag.Int("port", int(c.Port), "Port to run the server on")
	flag.StringVar(&c.RootDir, "root", c.RootDir, "Root directory containing markdown files")
	flag.StringVar(&c.RootDir, "r", c.RootDir, "Root directory (shorthand)")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Log level (debug, info, warn, error)")
	flag.StringVar(&c.StorageURL, "storage-url", c.StorageURL,
		"Blob storage URL (e.g., file:///path, s3://bucket/prefix, gs://bucket/prefix)")
	flag.BoolVar(&c.CacheEnabled, "cache", c.CacheEnabled, "Enable response caching")
	flag.BoolVar(&c.DevMode, "dev", c.DevMode, "Development mode (disables caching)")
	flag.DurationVar(&c.Timeout, "timeout", c.Timeout, "Request timeout")

	flag.Parse()

	if *portFlag > 0 && *portFlag <= 65535 {
		c.Port = uint16(*portFlag)
	}
}

// applyEnvironmentOverrides applies environment variable values to the config.
func (c *Config) applyEnvironmentOverrides() {
	c.applyEnvUint16("DYNAMIC_MARKDOWN_PORT", func(v uint16) { c.Port = v })
	c.applyEnvString("DYNAMIC_MARKDOWN_ROOT", func(v string) { c.RootDir = v })
	c.applyEnvString("DYNAMIC_MARKDOWN_LOG_LEVEL", func(v string) { c.LogLevel = v })
	c.applyEnvString("DYNAMIC_MARKDOWN_STORAGE_URL", func(v string) { c.StorageURL = v })
	c.applyEnvBool("DYNAMIC_MARKDOWN_CACHE", func(v bool) { c.CacheEnabled = v })
	c.applyEnvBool("DYNAMIC_MARKDOWN_DEV", func(v bool) { c.DevMode = v })
	c.applyEnvDuration("DYNAMIC_MARKDOWN_TIMEOUT", func(v time.Duration) { c.Timeout = v })
	c.applyEnvString("DYNAMIC_MARKDOWN_SITE_NAME", func(v string) { c.SiteName = v })
}

func (c *Config) applyEnvUint16(key string, apply func(uint16)) {
	if val := os.Getenv(key); val != "" {
		if p, err := parseUint16(val); err == nil {
			apply(p)
		}
	}
}

func (c *Config) applyEnvString(key string, apply func(string)) {
	if val := os.Getenv(key); val != "" {
		apply(val)
	}
}

func (c *Config) applyEnvBool(key string, apply func(bool)) {
	if val := os.Getenv(key); val != "" {
		apply(parseBool(val))
	}
}

func (c *Config) applyEnvDuration(key string, apply func(time.Duration)) {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			apply(d)
		}
	}
}

// applyDerivedSettings applies settings derived from other configuration values.
func (c *Config) applyDerivedSettings() error {
	if c.DevMode {
		c.CacheEnabled = false
	}

	if c.StorageURL == "" && !filepath.IsAbs(c.RootDir) {
		abs, err := filepath.Abs(c.RootDir)
		if err != nil {
			return errors.Wrap(err, "resolve root directory")
		}

		c.RootDir = abs
	}

	if c.StorageURL == "" {
		c.RootDir = filepath.Clean(c.RootDir)
	}

	return nil
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
		slog.String("site_name", c.SiteName),
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
	fmt.Fprintf(&b, "  Site Name:    %s\n", c.SiteName)

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
