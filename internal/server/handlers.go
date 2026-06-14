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
	httputil "github.com/larsartmann/httputil"
)

type Server struct {
	repo        content.Repository
	searcher    content.Searchable
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
	searcher content.Searchable,
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
		httputil.Recovery(s.logger),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		securityHeadersMiddleware(),
		s.accessLogMiddleware(),
		httputil.Compression(httputil.DefaultCompressionConfig()),
	)

	return handler
}

func (s *Server) Shutdown() {
	s.rateLimiter.Stop()
	s.cache.Close()
}

func (s *Server) LiveReload() *LiveReload {
	return s.liveReload
}

func (s *Server) handleContentOr404(w http.ResponseWriter, r *http.Request) {
	s.handleContentByPath(w, r, r.URL.Path)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		jsonKeyStatus:    jsonStatusHealthy,
		jsonKeyVersion:   version.Version,
		jsonKeyCommit:    version.Commit,
		jsonKeyBuildDate: version.BuildDate,
		jsonKeyTimestamp: time.Now().UTC(),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !s.rateLimiter.checkRateLimit(ip) {
		s.logger.Warn("rate limit exceeded for refresh endpoint", "client_ip", ip)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			jsonKeyStatus:    jsonStatusError,
			jsonKeyMessage:   "rate limit exceeded: too many refresh requests",
			jsonKeyLimit:     "10 requests per minute per IP",
			jsonKeyTimestamp: time.Now().UTC(),
		})

		return
	}

	s.logger.Info("manual refresh requested", "client_ip", ip)

	result := s.repo.Refresh()
	if !result.Success {
		s.logger.Error("failed to refresh repository", "error", result.Error)
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			jsonKeyStatus:    jsonStatusError,
			jsonKeyMessage:   "failed to refresh content repository",
			jsonKeyError:     result.Error,
			jsonKeyTimestamp: time.Now().UTC(),
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
		jsonKeyStatus:     jsonStatusSuccess,
		jsonKeyMessage:    "content repository refreshed",
		jsonKeyLastMod:    result.LastModified.UTC(),
		jsonKeyTotalFiles: result.TotalFiles,
		jsonKeyTotalDirs:  result.TotalDirs,
		jsonKeyDuration:   result.Duration,
		jsonKeyTimestamp:  time.Now().UTC(),
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
		// cleanPath is constructed from the request path with a forced leading
		// "/" and stripped of any leading slashes, guaranteeing a same-host
		// relative path. Safe to redirect.
		//nolint:gosec // G110: redirect target is a same-host path, not user-controlled URL
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
				// rawFile.Content is read from server-side storage (filesystem
				// or blob) and the Content-Type header is set from the file's
				// actual MIME type. Not a user-controlled response body.
				//nolint:gosec // G107: raw file content from repository, MIME type set explicitly
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
