.PHONY: build run test test-one test-cover lint fmt vet clean install check

# Binary name
BINARY_NAME=todoist-tui
BINARY_PATH=bin/$(BINARY_NAME)

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build the binary
build:
	$(GOBUILD) -o $(BINARY_PATH) ./cmd/todoist-tui

# Run in development mode
run:
	$(GORUN) ./cmd/todoist-tui

# Run all tests
test:
	$(GOTEST) -v ./...

# Run a single test (usage: make test-one TEST=TestFunctionName)
test-one:
	$(GOTEST) -v -run $(TEST) ./...

# Run a single test in a specific package (usage: make test-pkg PKG=./internal/api TEST=TestGetTasks)
test-pkg:
	$(GOTEST) -v -run $(TEST) $(PKG)

# Run tests with coverage
test-cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run golangci-lint
lint:
	golangci-lint run

# Format code
fmt:
	$(GOFMT) ./...
	@command -v goimports > /dev/null && goimports -w . || echo "goimports not installed, skipping"

# Vet code
vet:
	$(GOVET) ./...

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Install binary to $GOPATH/bin
install:
	$(GOCMD) install ./cmd/todoist-tui

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Run all checks before commit
check: fmt vet test

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_PATH)-linux-amd64 ./cmd/todoist-tui
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_PATH)-darwin-amd64 ./cmd/todoist-tui
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BINARY_PATH)-darwin-arm64 ./cmd/todoist-tui
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_PATH)-windows-amd64.exe ./cmd/todoist-tui

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  run         - Run in development mode"
	@echo "  test        - Run all tests"
	@echo "  test-one    - Run a single test (TEST=TestName)"
	@echo "  test-pkg    - Run a test in a package (PKG=./path TEST=TestName)"
	@echo "  test-cover  - Run tests with coverage report"
	@echo "  lint        - Run golangci-lint"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  check       - Run fmt, vet, and test"
	@echo "  build-all   - Build for multiple platforms"
