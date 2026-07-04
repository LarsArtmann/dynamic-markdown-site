package server

import (
	"net/http"

	templ "github.com/a-h/templ"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) handle404(w http.ResponseWriter, r *http.Request) {
	requestPath := r.URL.Path
	suggestions := s.getPathSuggestions(requestPath)

	s.renderNotFound(w, r, requestPath, suggestions)
}

func (s *Server) getPathSuggestions(requestedPath string) []domain.SuggestedPath {
	paths := s.repo.AllPaths()

	return findSuggestions(requestedPath, paths, 5)
}

func (s *Server) renderNotFound(
	w http.ResponseWriter, r *http.Request, requestPath string, suggestions []domain.SuggestedPath,
) {
	props := templates.ErrorViewProps{
		Title:       "Page Not Found",
		Message:     "The page you're looking for doesn't exist in this dimension.",
		StatusCode:  404,
		RequestPath: requestPath,
		Suggestions: suggestions,
		SiteName:    s.siteName,
	}

	component := templates.ErrorView(props)
	s.renderComponent(w, r, component, http.StatusNotFound, "404 page")
}

func (s *Server) handle500(w http.ResponseWriter, r *http.Request) {
	s.renderError(w, r, 500, "Internal Server Error",
		"Something went wrong in the cyber realm. Please try again later.")
}

func (s *Server) renderError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	title, message string,
) {
	props := templates.ErrorViewProps{
		Title:       title,
		Message:     message,
		StatusCode:  statusCode,
		RequestPath: "",
		Suggestions: nil,
		SiteName:    s.siteName,
	}

	component := templates.ErrorView(props)
	s.renderComponent(w, r, component, statusCode, "error page")
}

func (s *Server) renderComponent(w http.ResponseWriter, r *http.Request, component templ.Component,
	statusCode int, context string,
) {
	w.Header().Set(headerContentType, "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	err := component.Render(r.Context(), w)
	if err != nil {
		s.logger.Error("failed to render "+context, "status", statusCode, "error", err)

		switch statusCode {
		case http.StatusOK:
			s.handle500(w, r)
		case http.StatusInternalServerError:
			w.Header().Set(headerContentType, "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("Internal Server Error"))
		}
	}
}
