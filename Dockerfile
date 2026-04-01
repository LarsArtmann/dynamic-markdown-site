# Multi-stage build for dynamic-markdown-site
# OCI Image Specification v1.1 compliant
# SPDX: Proprietary

# =============================================================================
# STAGE 1: Builder
# =============================================================================
# Pin exact versions for reproducibility
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache ca-certificates git file

# Build arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Install templ CLI - PINNED VERSION for reproducibility
RUN go install github.com/a-h/templ/cmd/templ@v0.3.1001

# Set working directory
WORKDIR /build

# =============================================================================
# Dependency caching - use BuildKit cache mount for speed
# =============================================================================
# Copy dependency files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies into cache mount
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

# =============================================================================
# Build stage
# =============================================================================
COPY . .

# Generate templ templates
RUN templ generate

# Build static binary with version information
# - CGO_ENABLED=0: disable CGO for static linking
# - -tags netgo: use pure Go net resolver
# - -ldflags: inject version, commit, date for /health endpoint
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -tags netgo \
    -ldflags="\
        -w -s \
        -extldflags=-static \
        -X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${VERSION} \
        -X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${COMMIT} \
        -X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=${BUILD_DATE}" \
    -trimpath \
    -o dynamic-markdown-site \
    ./cmd/dynamic-markdown-site

# Verify binary is statically linked
RUN file dynamic-markdown-site && \
    (file dynamic-markdown-site | grep -q "statically linked" || exit 1)

# =============================================================================
# STAGE 2: Runtime (distroless Debian 13)
# =============================================================================
# Pin exact version - do not use floating tags
FROM gcr.io/distroless/static-debian13:nonroot@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39

# Re-declare ARGs (they don't persist across FROM stages)
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# OCI Image Specification v1.1 labels
# See: https://specs.opencontainers.org/image-spec/
LABEL org.opencontainers.image.authors="Lars Artmann <lars@larsartmann.com>" \
      org.opencontainers.image.source="https://github.com/larsartmann/dynamic-markdown-site" \
      org.opencontainers.image.description="Type-safe, high-performance markdown-to-website converter with syntax highlighting, diagrams, and full-text search" \
      org.opencontainers.image.licenses="Proprietary" \
      org.opencontainers.image.title="dynamic-markdown-site" \
      org.opencontainers.image.vendor="larsartmann" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.artifact.created="${BUILD_DATE}" \
      org.opencontainers.artifact.description="Markdown to website converter" \
      org.opencontainers.artifact.type="application/vnd.go" \
      org.opencontainers.artifact.version="${VERSION}" \
      org.opencontainers.software.spec.version="1.0" \
      org.opencontainers.image.name="dynamic-markdown-site" \
      maintainer="Lars Artmann <lars@larsartmann.com>"

# Set working directory
WORKDIR /app

# Copy the compiled binary (static assets embedded via //go:embed)
COPY --link --chown=nonroot:nonroot --from=builder /build/dynamic-markdown-site .

# Expose port
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

# Healthcheck omitted - distroless/static has no shell or utilities.
# Use external health checks: k8s livenessProbe, docker-compose healthcheck, etc.
# Example: curl http://localhost:8080/health

# Entrypoint using exec form (required for proper signal handling)
ENTRYPOINT ["/app/dynamic-markdown-site"]

# Default flags - root path points to /content volume
CMD ["-root", "/content", "-port", "8080", "-cache"]
