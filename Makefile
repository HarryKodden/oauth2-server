.PHONY: build run test clean docker-build docker-run test-flow test-integration benchmark coverage-html security tidy check

# Build the application
build:
	@echo "🔨 Building OAuth2 server..."
	@mkdir -p bin
	go build -ldflags "-s -w" -o bin/oauth2-server cmd/server/main.go
	@echo "✅ Build completed: bin/oauth2-server"

# Check for compilation errors without building
check:
	@echo "🔍 Checking for compilation errors..."
	go build -o /dev/null cmd/server/main.go
	@echo "✅ No compilation errors found"

# Run the application
run:
	@echo "🚀 Starting OAuth2 server..."
	go run cmd/server/main.go

# Run with live reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "🔄 Starting development server with live reload..."
	air

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -cover ./...

# Tidy dependencies
tidy:
	@echo "🧹 Tidying dependencies..."
	go mod tidy
	go mod download

# Clean build artifacts
clean:
	@echo "🗑️ Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Lint code
lint:
	@echo "🔍 Linting code..."
	golangci-lint run

# Format code
fmt:
	@echo "✨ Formatting code..."
	go fmt ./...

# Docker build
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t oauth2-server .

# Docker run
docker-run:
	@echo "🐳 Starting Docker containers..."
	docker-compose up

# Docker clean
docker-clean:
	@echo "🐳 Cleaning Docker containers..."
	docker-compose down -v

# Run specific test
test-flow:
	@echo "🧪 Running flow tests..."
	go test -v ./internal/flows/...

# Run integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	go test -v -tags=integration ./...

# Benchmark tests
benchmark:
	@echo "⚡ Running benchmarks..."
	go test -bench=. ./...

# Generate test coverage report
coverage-html:
	@echo "📈 Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"

# Check for security vulnerabilities
security:
	@echo "🔒 Checking for security vulnerabilities..."
	gosec ./...

# Install development dependencies
install-deps:
	@echo "📦 Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/cosmtrek/air@latest

# Quick check - format, tidy, check compilation
quick-check: fmt tidy check
	@echo "✅ Quick check completed"

# Full check - format, tidy, lint, test, security
full-check: fmt tidy lint test security
	@echo "✅ Full check completed"

# Build for multiple platforms
build-all:
	@echo "🔨 Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/oauth2-server-linux-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o bin/oauth2-server-darwin-amd64 cmd/server/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w" -o bin/oauth2-server-darwin-arm64 cmd/server/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o bin/oauth2-server-windows-amd64.exe cmd/server/main.go
	@echo "✅ Multi-platform build completed"

# Validate configuration
validate-config:
	@echo "🔍 Validating configuration..."
	go run cmd/server/main.go --validate-config
	@echo "✅ Configuration is valid"

# Generate example config
generate-config:
	@echo "📝 Generating example configuration..."
	@mkdir -p configs
	@cp config.yaml configs/config.example.yaml
	@echo "✅ Example configuration generated: configs/config.example.yaml"

# Show help
help:
	@echo "🔐 OAuth2 Server - Available Commands:"
	@echo ""
	@echo "Development:"
	@echo "  make build       - Build the application"
	@echo "  make run         - Run the application"
	@echo "  make dev         - Run with live reload"
	@echo "  make check       - Check for compilation errors"
	@echo ""
	@echo "Testing:"
	@echo "  make test        - Run all tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make test-flow   - Run flow tests only"
	@echo "  make benchmark   - Run benchmark tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt         - Format code"
	@echo "  make lint        - Lint code"
	@echo "  make security    - Security scan"
	@echo "  make quick-check - Format, tidy, check"
	@echo "  make full-check  - Complete validation"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build - Build Docker image"
	@echo "  make docker-run   - Run with Docker Compose"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make tidy        - Tidy dependencies"
	@echo "  make install-deps - Install dev dependencies"
	@echo "  make validate-config - Validate configuration"
	@echo "  make generate-config - Generate example configuration"