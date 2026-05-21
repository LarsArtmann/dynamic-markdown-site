package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/version"
)

type Server struct {
	repo        content.Repository
	searcher    *content.Searcher
	renderer    domain.Renderer
	logger      *slog.Logger
	rateLimiter *rateLimiter
	cache       *cache.HTMLCache
	liveReload  *LiveReload
	devMode     bool
	siteName    string
}

func NewServer(
	repo content.Repository,
	searcher *content.Searcher,
	log *slog.Logger,
	htmlCache *cache.HTMLCache,
	renderer domain.Renderer,
	devMode bool,
	siteName string,
) *Server {
	rl := newRateLimiter(10, time.Minute)
	lr := NewLiveReload(log)

	return &Server{
		repo:        repo,
		searcher:    searcher,
		renderer:    renderer,
		logger:      log,
		rateLimiter: rl,
		cache:       htmlCache,
		liveReload:  lr,
		devMode:     devMode,
		siteName:    siteName,
	}
}

// Handler returns the fully configured HTTP handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /robots.txt", s.handleRobotsTxt)
	mux.HandleFunc("GET /sitemap.xml", s.handleSitemapXML)
	mux.HandleFunc("GET /refresh", s.handleRefresh)
	mux.HandleFunc("POST /refresh", s.handleRefresh)
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /{$}", s.handleRoot)
	mux.HandleFunc("GET /api/live-reload", s.liveReload.handleSSE)
	mux.HandleFunc("GET /static/", s.serveStaticFile)
	mux.HandleFunc("GET /", s.handleContentOr404)

	var handler http.Handler = mux
	handler = chain(
		handler,
		s.accessLogMiddleware(),
		securityHeadersMiddleware(),
		requestIDMiddleware(),
	)

	return handler
}

func (s *Server) Shutdown() {
	s.rateLimiter.Stop()
}

func (s *Server) LiveReload() *LiveReload {
	return s.liveReload
}

func (s *Server) handleContentOr404(w http.ResponseWriter, r *http.Request) {
	s.handleContentByPath(w, r, r.URL.Path)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "healthy",
		"version":    version.Version,
		"commit":     version.Commit,
		"build_date": version.BuildDate,
		"timestamp":  time.Now().UTC(),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.rateLimiter.checkRateLimit(ip) {
		s.logger.Warn("rate limit exceeded for refresh endpoint", "client_ip", ip)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"status":    "error",
			"message":   "rate limit exceeded: too many refresh requests",
			"limit":     "10 requests per minute per IP",
			"timestamp": time.Now().UTC(),
		})

		return
	}

	s.logger.Info("manual refresh requested", "client_ip", ip)

	result := s.repo.Refresh()
	if !result.Success {
		s.logger.Error("failed to refresh repository", "error", result.Error)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"status":    "error",
			"message":   "failed to refresh content repository",
			"error":     result.Error,
			"timestamp": time.Now().UTC(),
		})

		return
	}

	s.cache.InvalidateAll()

	s.logger.Info(
		"repository refreshed successfully",
		"last_modified", result.LastModified,
		"total_files", result.TotalFiles,
		"total_dirs", result.TotalDirs,
		"duration", result.Duration,
	)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "success",
		"message":       "content repository refreshed",
		"last_modified": result.LastModified.UTC(),
		"total_files":   result.TotalFiles,
		"total_dirs":    result.TotalDirs,
		"duration":      result.Duration,
		"timestamp":     time.Now().UTC(),
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	root, err := s.repo.Root()
	if err != nil {
		s.logger.Error("failed to get root", "error", err, "path", "/")
		s.handle500(w, r)

		return
	}

	s.renderDirectory(w, r, root)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	if query == "" {
		s.renderSearch(w, r, query, nil)

		return
	}

	results, err := s.searcher.Search(query)
	if err != nil {
		s.logger.Error("search failed", "query", query, "error", err, "path", "/search")
		s.handle500(w, r)

		return
	}

	s.renderSearch(w, r, query, results)
}

func (s *Server) handleContentByPath(w http.ResponseWriter, r *http.Request, filepath string) {
	if before, ok := strings.CutSuffix(filepath, ".md"); ok {
		cleanPath := "/" + strings.TrimLeft(
			before,
			"/",
		)
		http.Redirect(w, r, cleanPath, http.StatusMovedPermanently)

		return
	}

	urlPath, err := domain.NewURLPath(filepath)
	if err != nil {
		s.logger.Warn("invalid path requested", "path", filepath, "error", err)
		s.handle404(w, r)

		return
	}

	node, err := s.repo.Get(urlPath)
	if err != nil {
		if errors.Is(err, content.ErrContentNotFound) {
			if rawFile, rawErr := s.repo.GetRaw(urlPath); rawErr == nil {
				w.Header().Set("Content-Type", rawFile.ContentType)
				w.Header().Set("Cache-Control", "public, max-age=86400")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(rawFile.Content)

				return
			}

			s.logger.Debug("content not found", "path", urlPath)
			s.handle404(w, r)

			return
		}

		s.logger.Error("failed to get content", "path", urlPath, "error", err)
		s.handle500(w, r)

		return
	}

	switch n := node.(type) {
	case *domain.DirectoryNode:
		s.renderDirectory(w, r, n)
	case *domain.FileNode:
		s.renderFile(w, r, n)
	default:
		s.logger.Error("unknown node type", "type", fmt.Sprintf("%T", node), "path", urlPath)
		s.handle500(w, r)
	}
}
