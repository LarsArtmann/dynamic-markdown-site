FROM golang:1.27-alpine AS builder

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
# Keep in sync with TEMPL_VERSION in test.yml and the templ version in go.mod.
ARG TEMPL_VERSION=v0.3.1020

ENV CGO_ENABLED=0 \
    GOEXPERIMENT=jsonv2

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY templates ./templates

# Generated *_templ.go files are gitignored; every build generates them
# (nix preBuild, CI, and here).
RUN go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION} \
    && templ generate

RUN go build -trimpath \
    -tags netgo,osusergo \
    -ldflags "-s -w -X github.com/larsartmann/dynamic-markdown-site/internal/version.Version=${VERSION} -X github.com/larsartmann/dynamic-markdown-site/internal/version.Commit=${COMMIT} -X github.com/larsartmann/dynamic-markdown-site/internal/version.BuildDate=${BUILD_DATE}" \
    -o /out/dynamic-markdown-site ./cmd/dynamic-markdown-site

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /out/dynamic-markdown-site /app/dynamic-markdown-site

EXPOSE 8080

USER 65532:65532

ENV PORT=8080 \
    DYNAMIC_MARKDOWN_PORT=8080 \
    DYNAMIC_MARKDOWN_LOG_LEVEL=info \
    DYNAMIC_MARKDOWN_CACHE=true \
    DYNAMIC_MARKDOWN_ROOT=/content

VOLUME ["/content"]

# Distroless images have no shell, curl, or wget. The binary implements the
# healthcheck subcommand which probes /health and exits 0 on a 200 response.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["/app/dynamic-markdown-site", "healthcheck", "--addr", "localhost:8080"]

ENTRYPOINT ["/app/dynamic-markdown-site"]
CMD ["-root", "/content", "-port", "8080", "-cache"]
