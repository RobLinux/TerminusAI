.PHONY: build install clean test run help

# Default target
all: build

# Build the application
build:
	@echo "Building terminusai..."
	go build -o terminusai.exe ./cmd/terminusai

# Build for different platforms
build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o terminusai-windows-amd64.exe ./cmd/terminusai

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o terminusai-linux-amd64 ./cmd/terminusai

build-macos:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o terminusai-darwin-amd64 ./cmd/terminusai

# Build all platforms
build-all: build-windows build-linux build-macos

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f terminusai.exe terminusai-*

# Run the application with help
run:
	go run ./cmd/terminusai --help

# Install the application to GOPATH/bin
install:
	@echo "Installing terminusai..."
	go install ./cmd/terminusai

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  build-all    - Build for all platforms"
	@echo "  deps         - Install dependencies"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run application with help"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  help         - Show this help message"