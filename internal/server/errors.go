// Package server provides HTTP server implementation and request handling.
package server

import (
	"net/http"

	templ "github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) handle404(c *gin.Context) {
	c.Status(http.StatusNotFound)

	requestPath := c.Request.URL.Path
	suggestions := s.getPathSuggestions(requestPath)

	s.renderNotFound(c, requestPath, suggestions)
}

// getPathSuggestions returns path suggestions based on similarity to the requested path.
func (s *Server) getPathSuggestions(requestedPath string) []SuggestedPath {
	paths := s.repo.AllPaths()

	return findSuggestions(requestedPath, paths, 5)
}

func (s *Server) renderNotFound(c *gin.Context, requestPath string, suggestions []SuggestedPath) {
	props := templates.ErrorViewProps{
		Title:       "Page Not Found",
		Message:     "The page you're looking for doesn't exist in this dimension.",
		StatusCode:  404,
		RequestPath: requestPath,
		Suggestions: convertToTemplateSuggestions(suggestions),
	}

	component := templates.ErrorView(props)
	s.renderComponent(c, component, http.StatusNotFound, "404 page")
}

// convertToTemplateSuggestions converts server suggestions to template suggestions.
func convertToTemplateSuggestions(suggestions []SuggestedPath) []templates.SuggestedPath {
	result := make([]templates.SuggestedPath, len(suggestions))

	for i, s := range suggestions {
		result[i] = templates.SuggestedPath{
			Path:  s.Path,
			Title: s.Title,
			Score: s.Score,
		}
	}

	return result
}

func (s *Server) handle500(c *gin.Context) {
	c.Status(http.StatusInternalServerError)

	s.renderError(c, 500, "Internal Server Error",
		"Something went wrong in the cyber realm. Please try again later.")
}

func (s *Server) renderError(c *gin.Context, statusCode int, title, message string) {
	props := templates.ErrorViewProps{
		Title:      title,
		Message:    message,
		StatusCode: statusCode,
	}

	component := templates.ErrorView(props)
	s.renderComponent(c, component, statusCode, "error page")
}

func (s *Server) renderComponent(c *gin.Context, component templ.Component,
	statusCode int, context string,
) {
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		s.logger.Error("failed to render "+context, "status", statusCode, "error", err)
		switch statusCode {
		case http.StatusOK:
			// When rendering normal templates fails, we need to handle 500
			s.handle500(c)
		case http.StatusInternalServerError:
			// When rendering error page fails, just send string
			c.String(statusCode, "Internal Server Error")
		}
	}
}
