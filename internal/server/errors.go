package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) handle404(c *gin.Context) {
	c.Status(http.StatusNotFound)

	props := templates.ErrorViewProps{
		Title:      "Page Not Found",
		Message:    "The page you're looking for doesn't exist in this dimension.",
		StatusCode: 404,
	}

	component := templates.ErrorView(props)

	err := component.Render(c.Request.Context(), c.Writer)
	if err != nil {
		s.logger.Error("failed to render 404", "error", err)
	}
}

func (s *Server) handle500(c *gin.Context) {
	c.Status(http.StatusInternalServerError)

	props := templates.ErrorViewProps{
		Title:      "Internal Server Error",
		Message:    "Something went wrong in the cyber realm. Please try again later.",
		StatusCode: 500,
	}

	component := templates.ErrorView(props)

	err := component.Render(c.Request.Context(), c.Writer)
	if err != nil {
		s.logger.Error("failed to render 500", "error", err)
		c.String(http.StatusInternalServerError, "Internal Server Error")
	}
}
