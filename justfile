# Justfile for dynamic-markdown-site

# Install locally using go install
install-local:
    go install ./cmd/dynamic-markdown-site

# Run in development mode
run-dev:
    go run ./cmd/dynamic-markdown-site -dev -root ./content

# Run tests
test:
    go test ./... -cover

# Run tests verbosely
test-v:
    go test ./... -v

# Run linter
lint:
    golangci-lint run ./...

# Generate templ templates
generate:
    templ generate

# Build binary
build:
    go build -o dynamic-markdown-site ./cmd/dynamic-markdown-site

# Clean build artifacts
clean:
    rm -f dynamic-markdown-site

# Run benchmarks
bench:
    go test ./... -bench=. -benchmem
