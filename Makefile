.PHONY: help test fmt vet lint tidy build example clean

# Default target
help:
	@echo "Available targets:"
	@echo "  test       - Run tests"
	@echo "  fmt        - Format code"
	@echo "  vet        - Run go vet"
	@echo "  lint       - Run golangci-lint (if available)"
	@echo "  tidy       - Tidy dependencies"
	@echo "  build      - Build examples"
	@echo "  example    - Run basic example"
	@echo "  clean      - Clean build artifacts"
	@echo "  all        - Run fmt, vet, tidy, and test"

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run golangci-lint if available
lint:
	@which golangci-lint > /dev/null && golangci-lint run ./... || echo "golangci-lint not installed"

# Tidy dependencies
tidy:
	go mod tidy

# Build examples
build:
	cd examples/basic && go build -o app main.go

# Run basic example
example:
	@if [ -z "$$OTEL_EXPORTER_OTLP_ENDPOINT" ]; then \
		echo "Error: OTEL_EXPORTER_OTLP_ENDPOINT not set"; \
		echo "Example: export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317"; \
		exit 1; \
	fi
	cd examples/basic && go run main.go

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	rm -f examples/*/app
	find . -name "*.test" -type f -delete

# Run all checks
all: fmt vet tidy test
	@echo "All checks passed!"
