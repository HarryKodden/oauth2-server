.PHONY: build run test clean docker-build docker-run test-flow test-integration benchmark coverage-html security tidy check examples run-examples clean-examples examples-auth-code examples-client-creds examples-device examples-token-exchange examples-all run-examples-auth run-examples-client run-examples-device run-examples-token

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

# Code quality targets
fmt:
	@echo "🎨 Formatting Go code..."
	gofmt -s -w .
	@echo "✅ Code formatted successfully"

vet:
	@echo "🔍 Running go vet..."
	go vet ./...
	@echo "✅ go vet completed"

staticcheck:
	@echo "🔍 Running staticcheck..."
	@if ! command -v staticcheck >/dev/null 2>&1; then \
		echo "Installing staticcheck..."; \
		go install honnef.co/go/tools/cmd/staticcheck@latest; \
	fi
	@echo "Running staticcheck with full Go bin path..."
	$(shell go env GOPATH)/bin/staticcheck ./...
	@echo "✅ staticcheck completed"

# Alternative staticcheck target that uses go run instead
staticcheck-alt:
	@echo "🔍 Running staticcheck (alternative method)..."
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...
	@echo "✅ staticcheck completed"

# Enhanced lint target with better error handling
lint: fmt vet
	@echo "🔍 Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	#elif [ -f "$(shell go env GOPATH)/bin/staticcheck" ]; then \
	#	$(shell go env GOPATH)/bin/staticcheck ./...; \
	#else \
	#	echo "Installing staticcheck..."; \
	#	go install honnef.co/go/tools/cmd/staticcheck@latest; \
	#	$(shell go env GOPATH)/bin/staticcheck ./...; \
	#fi
	@echo "✅ All linting completed"

# Install all development tools with proper PATH setup
install-deps:
	@echo "📦 Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/cosmtrek/air@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "📍 Tools installed in: $(shell go env GOPATH)/bin"
	@echo "💡 Make sure $(shell go env GOPATH)/bin is in your PATH"
	@echo "   Add this to your shell profile:"
	@echo "   export PATH=\"$(shell go env GOPATH)/bin:\$$PATH\""

# Check if tools are properly installed
check-tools:
	@echo "🔧 Checking development tools..."
	@echo "Go version: $(shell go version)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@echo "GOBIN: $(shell go env GOBIN)"
	@echo ""
	@echo "Checking tool availability:"
	@if command -v staticcheck >/dev/null 2>&1; then \
		echo "✅ staticcheck: $(shell which staticcheck)"; \
	#elif [ -f "$(shell go env GOPATH)/bin/staticcheck" ]; then \
	#	echo "⚠️  staticcheck: $(shell go env GOPATH)/bin/staticcheck (not in PATH)"; \
	#else \
	#	echo "❌ staticcheck: not installed"; \
	#fi
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "✅ golangci-lint: $(shell which golangci-lint)"; \
	#elif [ -f "$(shell go env GOPATH)/bin/golangci-lint" ]; then \
	#	echo "⚠️  golangci-lint: $(shell go env GOPATH)/bin/golangci-lint (not in PATH)"; \
	#else \
	#	echo "❌ golangci-lint: not installed"; \
	#fi
	@if command -v gosec >/dev/null 2>&1; then \
		echo "✅ gosec: $(shell which gosec)"; \
	#elif [ -f "$(shell go env GOPATH)/bin/gosec" ]; then \
	#	echo "⚠️  gosec: $(shell go env GOPATH)/bin/gosec (not in PATH)"; \
	#else \
	#	echo "❌ gosec: not installed"; \
	#fi

# Enhanced security check with proper path handling
security:
	@echo "🔒 Checking for security vulnerabilities..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	#elif [ -f "$(shell go env GOPATH)/bin/gosec" ]; then \
	#	$(shell go env GOPATH)/bin/gosec ./...; \
	#else \
	#	echo "Installing gosec..."; \
	#	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
	#	$(shell go env GOPATH)/bin/gosec ./...; \
	#fi

# Enhanced golangci-lint target
golangci-lint:
	@echo "🔍 Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	#elif [ -f "$(shell go env GOPATH)/bin/golangci-lint" ]; then \
	#	$(shell go env GOPATH)/bin/golangci-lint run; \
	#else \
	#	echo "Installing golangci-lint..."; \
	#	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	#	$(shell go env GOPATH)/bin/golangci-lint run; \
	#fi
	@echo "✅ golangci-lint completed"

# Comprehensive lint target using golangci-lint (includes staticcheck)
lint-comprehensive: fmt vet golangci-lint
	@echo "✅ Comprehensive linting completed"

# Fix PATH issues by setting up proper Go environment
setup-env:
	@echo "🔧 Setting up Go development environment..."
	@echo "Current GOPATH: $(shell go env GOPATH)"
	@echo "Current PATH: $$PATH"
	@echo ""
	@echo "To fix PATH issues, add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
	@echo "export PATH=\"$(shell go env GOPATH)/bin:\$$PATH\""
	@echo ""
	@echo "Or run this command to add it temporarily:"
	@echo "export PATH=\"$(shell go env GOPATH)/bin:\$$PATH\""

# Update .PHONY to include new targets
.PHONY: fmt vet staticcheck staticcheck-alt lint golangci-lint lint-comprehensive fix-imports pre-commit install-deps check-tools security setup-env

# Update existing targets to use the new patterns
# Pre-commit checks with better tool handling
pre-commit: fmt vet
	@echo "🔍 Running pre-commit checks..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	#elif [ -f "$(shell go env GOPATH)/bin/staticcheck" ]; then \
	#	$(shell go env GOPATH)/bin/staticcheck ./...; \
	#else \
	#	echo "Installing staticcheck..."; \
	#	go install honnef.co/go/tools/cmd/staticcheck@latest; \
	#	$(shell go env GOPATH)/bin/staticcheck ./...; \
	#fi
	@$(MAKE) test
	@echo "✅ Pre-commit checks completed"

# Full check with comprehensive linting
full-check: fmt tidy lint-comprehensive test security
	@echo "✅ Full check completed"