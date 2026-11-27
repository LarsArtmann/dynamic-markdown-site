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
	httputil "github.com/larsartmann/httputil"
	"github.com/larsartmann/httputil/endpoints"
)

type Server struct {
	repo              content.Repository
	searcher          content.Searchable
	renderer          domain.Renderer
	logger            *slog.Logger
	rateLimiter       *rateLimiter
	searchRateLimiter *rateLimiter
	cache             *cache.HTMLCache
	liveReload        *LiveReload
	healthMetrics     *HealthMetrics
	devMode           bool
	siteName          string
	startedAt         time.Time
}

func NewServer(
	repo content.Repository,
	searcher content.Searchable,
	log *slog.Logger,
	htmlCache *cache.HTMLCache,
	renderer domain.Renderer,
	healthMetrics *HealthMetrics,
	devMode bool,
	siteName string,
) *Server {
	rl := newRateLimiter(10, time.Minute)
	srl := newRateLimiter(searchRateLimit, time.Minute)
	lr := NewLiveReload(log)

	return &Server{
		repo:              repo,
		searcher:          searcher,
		renderer:          renderer,
		logger:            log,
		rateLimiter:       rl,
		searchRateLimiter: srl,
		cache:             htmlCache,
		liveReload:        lr,
		healthMetrics:     healthMetrics,
		devMode:           devMode,
		siteName:          siteName,
		startedAt:         time.Now(),
	}
}

// Handler returns the fully configured HTTP handler with all routes and middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	routes := endpoints.NewRegistry()

	// The declarations below ARE the site's route table: each one mounts its
	// handler AND records it for the GET /endpoints listing
	// (httputil/endpoints), so the listing cannot drift from what is served.
	mount(routes, mux, "GET /robots.txt", s.handleRobotsTxt, "robots.txt: all crawlers allowed, sitemap URL")
	mount(routes, mux, "GET /sitemap.xml", s.handleSitemapXML, "XML sitemap of every page and directory")
	mount(routes, mux, "GET /refresh", s.handleRefresh, "Reload content and clear the HTML cache (JSON; 10/min per IP)")
	mount(routes, mux, "POST /refresh", s.handleRefresh, "Reload content and clear the HTML cache (JSON; 10/min per IP)")
	mount(routes, mux, "GET /search", s.handleSearch, "Full-text search page (?q=, page, pageSize; 30/min per IP)")
	mount(routes, mux, "GET /cache/stats", s.handleCacheStats, "HTML cache size, hit, miss, and eviction counters (JSON)")
	mount(routes, mux, "GET /metrics", s.handleMetrics, "Prometheus text exposition: cache counters, uptime, health gauges")
	mount(routes, mux, "GET /{$}", s.handleRoot, "Root directory listing (the site's front page)")
	s.registerLiveReloadRoute(mux, routes)
	mount(routes, mux, "GET /static/", s.serveStaticFile, "Embedded static assets (CSS, favicon)")
	mount(routes, mux, "GET /", s.handleContentOr404, "Markdown page, directory listing, raw file, or 404 page by path")

	// The listing sits on the site mux, so it rides the site middleware chain
	// (security headers, compression, access log) like every other route.
	mustMount(endpoints.Register(mux, routes))

	var handler http.Handler = mux
	handler = chain(
		handler,
		httputil.Recovery(s.logger),
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
		securityHeadersMiddleware(),
		s.accessLogMiddleware(),
		s.responseTimeMiddleware(),
		httputil.Compression(httputil.DefaultCompressionConfig()),
	)

	return &registeredHandler{Handler: handler, routes: routes}
}

func (s *Server) Shutdown() {
	s.rateLimiter.Stop()
	s.searchRateLimiter.Stop()
	s.liveReload.Close()
	s.cache.Close()
}

func (s *Server) LiveReload() *LiveReload {
	return s.liveReload
}

func (s *Server) handleContentOr404(w http.ResponseWriter, r *http.Request) {
	s.handleContentByPath(w, r, r.URL.Path)
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
	ip := clientIP(r)
	if !s.searchRateLimiter.checkRateLimit(ip) {
		s.logger.Warn("rate limit exceeded for search endpoint", "client_ip", ip)
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			jsonKeyStatus:    jsonStatusError,
			jsonKeyMessage:   "rate limit exceeded: too many search requests",
			jsonKeyLimit:     fmt.Sprintf("%d requests per minute per IP", searchRateLimit),
			jsonKeyTimestamp: time.Now().UTC(),
		})

		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))

	if query == "" {
		s.renderSearch(w, r, query, nil, domain.NewSearchPagination(0, "", ""))

		return
	}

	results, err := s.searcher.Search(query)
	if err != nil {
		s.logger.Error("search failed", "query", query, "error", err, "path", "/search")
		s.handle500(w, r)

		return
	}

	pagination := domain.NewSearchPagination(
		len(results),
		r.URL.Query().Get("page"),
		r.URL.Query().Get("pageSize"),
	)

	s.renderSearch(w, r, query, results[pagination.Offset():pagination.End()], pagination)
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
				w.Header().Set(headerContentType, rawFile.ContentType)
				w.Header().Set("Cache-Control", "public, max-age=86400")
				w.WriteHeader(http.StatusOK)
				// rawFile.Content is read from server-side storage (filesystem
				// or blob) and the Content-Type header is set from the file's
				// actual MIME type. Not a user-controlled response body.
				//nolint:gosec // G107: raw file content from repository, MIME type set explicitly
				_, _ = w.Write(rawFile.Content) //nolint:erraudit // client disconnect mid-write is unrecoverable

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
