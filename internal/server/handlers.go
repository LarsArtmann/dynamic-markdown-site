package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/renderer"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
)

// Server holds all dependencies for HTTP handling.
type Server struct {
	repo        content.Repository
	searcher    *content.Searcher
	renderer    *renderer.GoldmarkRenderer
	logger      *slog.Logger
	rateLimiter *rateLimiter
	cache       *cache.HTMLCache
	liveReload  *LiveReload
	devMode     bool
	siteName    string
}

// NewServer creates a new HTTP server with dependencies.
func NewServer(
	repo content.Repository,
	searcher *content.Searcher,
	log *slog.Logger,
	cache *cache.HTMLCache,
	devMode bool,
	siteName string,
) *Server {
	rl := newRateLimiter(10, time.Minute)
	lr := NewLiveReload(log)

	return &Server{
		repo:        repo,
		searcher:    searcher,
		renderer:    renderer.NewGoldmarkRenderer(),
		logger:      log,
		rateLimiter: rl,
		cache:       cache,
		liveReload:  lr,
		devMode:     devMode,
		siteName:    siteName,
	}
}

// RegisterRoutes sets up all HTTP routes.
func (s *Server) RegisterRoutes(router *gin.Engine) {
	// Global middleware (order matters - first to last)
	router.Use(requestIDMiddleware())
	router.Use(securityHeadersMiddleware())
	router.Use(s.accessLogMiddleware())
	router.Use(s.staticAndContentMiddleware())
	router.GET("/health", s.handleHealth)
	router.GET("/robots.txt", s.handleRobotsTxt)
	router.GET("/refresh", s.handleRefresh)
	router.POST("/refresh", s.handleRefresh)
	router.GET("/search", s.handleSearch)
	router.GET("/", s.handleRoot)
	s.liveReload.RegisterHandler(router)
	router.NoRoute(s.handle404)
}

// LiveReload returns the LiveReload instance for external notification.
func (s *Server) LiveReload() *LiveReload {
	return s.liveReload
}

func (s *Server) staticAndContentMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/health" || path == "/refresh" ||
			path == "/search" || path == "/" ||
			path == "/robots.txt" {
			c.Next()

			return
		}

		if strings.HasPrefix(path, "/static/") {
			s.serveStaticFile(c)
			c.Abort()

			return
		}

		s.handleContentByPath(c, path)
		c.Abort()
	}
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":     "healthy",
		"version":    version.Version,
		"commit":     version.Commit,
		"build_date": version.BuildDate,
		"timestamp":  time.Now().UTC(),
	})
}

func (s *Server) handleRefresh(c *gin.Context) {
	clientIP := c.ClientIP()
	if !s.rateLimiter.checkRateLimit(clientIP) {
		s.logger.Warn("rate limit exceeded for refresh endpoint", "client_ip", clientIP)
		c.JSON(http.StatusTooManyRequests, gin.H{
			"status":    "error",
			"message":   "rate limit exceeded: too many refresh requests",
			"limit":     "10 requests per minute per IP",
			"timestamp": time.Now().UTC(),
		})

		return
	}

	s.logger.Info("manual refresh requested", "client_ip", clientIP)

	result := s.repo.Refresh()
	if !result.Success {
		s.logger.Error("failed to refresh repository", "error", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":    "error",
			"message":   "failed to refresh content repository",
			"error":     result.Error,
			"timestamp": time.Now().UTC(),
		})

		return
	}

	// Invalidate HTML cache when content is refreshed
	s.cache.InvalidateAll()

	s.logger.Info("repository refreshed successfully",
		"last_modified", result.LastModified,
		"total_files", result.TotalFiles,
		"total_dirs", result.TotalDirs,
		"duration", result.Duration,
	)

	c.JSON(http.StatusOK, gin.H{
		"status":        "success",
		"message":       "content repository refreshed",
		"last_modified": result.LastModified.UTC(),
		"total_files":   result.TotalFiles,
		"total_dirs":    result.TotalDirs,
		"duration":      result.Duration,
		"timestamp":     time.Now().UTC(),
	})
}

func (s *Server) handleRoot(c *gin.Context) {
	root, err := s.repo.Root()
	if err != nil {
		s.logger.Error("failed to get root", "error", err, "path", "/")
		s.handle500(c)

		return
	}

	s.renderDirectory(c, root)
}

func (s *Server) handleSearch(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))

	if query == "" {
		s.renderSearch(c, query, nil)

		return
	}

	results, err := s.searcher.Search(query)
	if err != nil {
		s.logger.Error("search failed", "query", query, "error", err, "path", "/search")
		s.handle500(c)

		return
	}

	s.renderSearch(c, query, results)
}

func (s *Server) handleContentByPath(c *gin.Context, filepath string) {
	urlPath, err := domain.NewURLPath(filepath)
	if err != nil {
		s.logger.Warn("invalid path requested", "path", filepath, "error", err)
		s.handle404(c)

		return
	}

	node, err := s.repo.Get(urlPath)
	if err != nil {
		if errors.Is(err, content.ErrContentNotFound) {
			s.logger.Debug("content not found", "path", urlPath)
			s.handle404(c)

			return
		}

		s.logger.Error("failed to get content", "path", urlPath, "error", err)
		s.handle500(c)

		return
	}

	switch n := node.(type) {
	case *domain.DirectoryNode:
		s.renderDirectory(c, n)
	case *domain.FileNode:
		s.renderFile(c, n)
	default:
		s.logger.Error("unknown node type", "type", fmt.Sprintf("%T", node), "path", urlPath)
		s.handle500(c)
	}
}
