# Multi-stage build for dynamic-markdown-site
# Follows Rolls-Royce Google Cloud IaC Automation Dockerfile requirements

# =============================================================================
# STAGE 1: Builder
# =============================================================================
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git

# Install templ CLI for template generation
RUN go install github.com/a-h/templ/cmd/templ@latest

# Set working directory
WORKDIR /build

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Generate templ templates
RUN templ generate

# Build static binary
# - CGO_ENABLED=0: disable CGO for static linking
# - -tags netgo: use pure Go net resolver
# - -extldflags '-static': static linking
# - -ldflags '-w -s': strip debug info and symbol table
# - -trimpath: remove all file system paths from the resulting executable
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -tags netgo \
    -ldflags='-w -s -extldflags=-static' \
    -trimpath \
    -o dynamic-markdown-site \
    ./cmd/dynamic-markdown-site

# =============================================================================
# STAGE 2: Runtime (distroless)
# =============================================================================
FROM gcr.io/distroless/static-debian13:nonroot

# OCI-compliant labels
LABEL org.opencontainers.image.source="https://github.com/larsartmann/dynamic-markdown-site" \
      org.opencontainers.image.description="Type-safe, high-performance markdown-to-website converter" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="dynamic-markdown-site" \
      org.opencontainers.image.vendor="larsartmann"

# Set working directory
WORKDIR /app

# Copy static assets (CSS, favicon, etc.)
COPY --chown=nonroot:nonroot internal/static/ ./internal/static/

# Copy the compiled binary
COPY --chown=nonroot:nonroot --from=builder /build/dynamic-markdown-site .

# Expose port (configurable via PORT env var, defaults to 8080)
EXPOSE 8080

# Run as non-root user (distroless:nonroot is UID 65532)
USER nonroot:nonroot

# Environment variables with sensible defaults
ENV PORT=8080 \
    DYNAMIC_MARKDOWN_PORT=8080 \
    DYNAMIC_MARKDOWN_LOG_LEVEL=info \
    DYNAMIC_MARKDOWN_CACHE=true \
    DYNAMIC_MARKDOWN_ROOT=/content

# Volume for markdown content (mount your content here)
VOLUME ["/content"]

# Entrypoint using exec form (required for proper signal handling)
ENTRYPOINT ["/app/dynamic-markdown-site"]

# Default flags - root path points to /content volume
CMD ["-root", "/content", "-port", "8080", "-cache"]
