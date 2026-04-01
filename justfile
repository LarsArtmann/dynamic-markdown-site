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

# Run tests with race detector
test-race:
    go test ./... -race -cover

# Run tests verbosely
test-v:
    go test ./... -v

# Run linter
lint:
    golangci-lint run ./...

# Fix formatting issues (golines)
fix:
    golines -w .

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

# Run everything needed before pushing (lint + test + race)
pre-push: lint test test-race
    @echo "All checks passed - safe to push"

# Generate templates then build
gen-build: generate build

# Run tests with coverage report
cover:
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"
