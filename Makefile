.PHONY: build run test clean lint install

# Binary name
BINARY=relay-mcp

# Build info
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-s -w \
    -X github.com/Narcoleptic-Fox/relay-mcp/internal/config.Version=$(VERSION) \
    -X github.com/Narcoleptic-Fox/relay-mcp/internal/config.Commit=$(COMMIT) \
    -X github.com/Narcoleptic-Fox/relay-mcp/internal/config.BuildTime=$(BUILD_TIME)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY).exe ./cmd/relay-mcp

# Run the server
run: build
	./$(BINARY).exe

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Clean build artifacts
clean:
	del $(BINARY).exe
	del coverage.out
	del coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy
