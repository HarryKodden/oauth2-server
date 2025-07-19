.PHONY: build run test clean help docker-build docker-run deps fmt vet dev demo ci well-known clients

# Variables
BINARY_NAME=oauth2-server
DOCKER_IMAGE=oauth2-server
PORT=8080

# Default target
help:
	@echo "OAuth2 Server with Device Code Flow & Token Exchange"
	@echo "==================================================="
	@echo "Available targets:"
	@echo "  build       - Build the OAuth2 server"
	@echo "  run         - Run the OAuth2 server"
	@echo "  test        - Run comprehensive OAuth2 flow tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run  - Run server in Docker container"
	@echo "  deps        - Install dependencies"
	@echo "  fmt         - Format Go code"
	@echo "  vet         - Run go vet"
	@echo "  dev         - Development build (fmt + vet + build)"
	@echo "  demo        - Start server and open web interface"
	@echo "  well-known  - Show OAuth2 well-known configuration"
	@echo "  clients     - Show registered client information"

# Build the OAuth2 server
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) .

# Run the OAuth2 server
run:
	@echo "Starting OAuth2 server on port $(PORT)..."
	@echo "Web interface: http://localhost:$(PORT)"
	@echo "Well-known config: http://localhost:$(PORT)/.well-known/oauth-authorization-server"
	@go run .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Format Go code
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Run comprehensive OAuth2 flow tests
test:
	@echo "Running comprehensive OAuth2 flow tests..."
	@echo "Testing: Authorization Code, Client Credentials, Refresh Token, Device Code Flow, Token Exchange"
	@./test_complete_flow.sh

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@go clean

# Build Docker image
docker-build:
	@echo "Building Docker image $(DOCKER_IMAGE)..."
	@docker build -t $(DOCKER_IMAGE) .

# Run Docker container
docker-run: docker-build
	@echo "Running OAuth2 server in Docker container on port $(PORT)..."
	@echo "Web interface: http://localhost:$(PORT)"
	@docker run -p $(PORT):$(PORT) $(DOCKER_IMAGE)

# Development target - format, vet, and build
dev: fmt vet build

# Demo target - build and run with browser
demo: build
	@echo "Starting OAuth2 server demo..."
	@echo "Server will start on http://localhost:$(PORT)"
	@echo "Press Ctrl+C to stop"
	@./$(BINARY_NAME) &
	@SERVER_PID=$$!; \
	sleep 2; \
	echo "Opening web interface..."; \
	open http://localhost:$(PORT) 2>/dev/null || echo "Visit http://localhost:$(PORT) in your browser"; \
	echo "Press Enter to stop server"; \
	read; \
	kill $$SERVER_PID

# CI target - format, vet, build, and test
ci: fmt vet build
	@echo "Starting server for CI testing..."
	@./$(BINARY_NAME) &
	@SERVER_PID=$$!; \
	sleep 5; \
	./test_complete_flow.sh; \
	TEST_RESULT=$$?; \
	kill $$SERVER_PID; \
	exit $$TEST_RESULT

# Show OAuth2 well-known configuration
well-known:
	@echo "Fetching OAuth2 well-known configuration..."
	@curl -s http://localhost:$(PORT)/.well-known/oauth-authorization-server | jq . || echo "Server not running or jq not installed"

# Show registered clients information
clients:
	@echo "Registered OAuth2 Clients:"
	@echo "=========================="
	@echo "Frontend Client (frontend-client):"
	@echo "  - Grant Types: authorization_code, refresh_token, device_code"
	@echo "  - Scopes: openid, profile, email, offline_access, api:read"
	@echo "  - Use Case: User-facing applications, device authorization"
	@echo ""
	@echo "Backend Client (backend-client):"
	@echo "  - Grant Types: client_credentials, token_exchange, refresh_token"
	@echo "  - Scopes: api:read, api:write"
	@echo "  - Use Case: Service-to-service communication, long-running processes"
	@echo ""
	@echo "Test User Credentials:"
	@echo "  - Username: john.doe"
	@echo "  - Password: password123"
