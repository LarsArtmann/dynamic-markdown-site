// Package server provides HTTP server implementation and request handling.
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) handle404(c *gin.Context) {
	c.Status(http.StatusNotFound)

	s.renderError(c, 404, "Page Not Found", "The page you're looking for doesn't exist in this dimension.")
}

func (s *Server) handle500(c *gin.Context) {
	c.Status(http.StatusInternalServerError)

	s.renderError(c, 500, "Internal Server Error", "Something went wrong in the cyber realm. Please try again later.")
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

func (s *Server) renderComponent(c *gin.Context, component templ.Component, statusCode int, context string) {
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		s.logger.Error("failed to render "+context, "status", statusCode, "error", err)
		if statusCode == http.StatusInternalServerError {
			c.String(statusCode, "Internal Server Error")
		}
	}
}
