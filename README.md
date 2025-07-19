# OAuth2 Server with Device Code Flow & Token Exchange

A comprehensive OAuth2/OIDC server implementation using the [Fosite](https://github.com/ory/fosite) library with support for:

- **📱 Device Code Flow (RFC 8628)** - For devices with limited input capabilities
- **🔄 Token Exchange (RFC 8693)** - For service-to-service token delegation
- **♻️ Refresh Tokens** - For long-running applications and token renewal
- **🎯 Two-Client Architecture** - Frontend and backend clients with clear separation of concerns
- **🌐 OpenID Connect** - Full OIDC compliance with comprehensive well-known configuration
- **🔒 JWT Tokens** - RS256 signed tokens with proper validation

## Features

### Frontend Client (`frontend-client`)
- **Device Code Flow** for CLI applications and IoT devices
- **Authorization Code Flow** for traditional web applications
- **OpenID Connect** scopes (`openid`, `profile`, `email`)
- **Refresh token** support for offline access
- **Multi-audience** tokens for service integration

### Backend Client (`backend-client`)
- **Client Credentials Flow** for service authentication
- **Token Exchange** for cross-service token delegation
- **Refresh token** capability for long-running processes
- **Service-specific** scopes (`api:read`, `api:write`)
- **Audience validation** for security

## Quick Start

### Using Make (Recommended)

```bash
# Build and run the server
make run

# Run comprehensive tests
make test

# Start demo with web interface
make demo

# Show client information
make clients

# Check well-known configuration
make well-known
```

### Manual Setup

1. Clone or download the project

2. Install dependencies:

```bash
go mod tidy
```

3. Build and run:

```bash
make build
./oauth2-server
```

The server will start on `http://localhost:8080`

## Testing & Usage

### Web Interface (Easiest)

1. Start the server: `make run`
2. Open: `http://localhost:8080`
3. Use the interactive web interface to test all OAuth2 flows
4. Login with: `username=john.doe`, `password=password123`

### 1. Device Code Flow Testing

Visit `http://localhost:8080` or use curl:

```bash
# Initiate device authorization
curl -X POST http://localhost:8080/device_authorization \
  -d "client_id=frontend-client" \
  -d "scope=openid profile email api:read"
```

This returns:

```json
{
  "device_code": "device_1234567890_abcdef",
  "user_code": "ABC123",
  "verification_uri": "http://localhost:8080/device",
  "verification_uri_complete": "http://localhost:8080/device?user_code=ABC123",
  "expires_in": 600,
  "interval": 5
}
```

### 2. User Authorization

1. Visit the `verification_uri_complete` URL
2. Enter credentials:
   - **Username:** `john.doe`
   - **Password:** `password123`

### 3. Poll for Token

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=device_1234567890_abcdef" \
  -d "client_id=frontend-client"
```

### 4. Token Exchange (Backend Client)

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "client_id=backend-client" \
  -d "client_secret=backend-client-secret" \
  -d "subject_token=<access_token_from_frontend_client>" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "audience=api-service"
```

### 5. Client Credentials Flow

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=client_credentials" \
  -d "client_id=backend-client" \
  -d "client_secret=backend-client-secret" \
  -d "scope=api:read api:write"
```

### 6. Refresh Tokens

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=refresh_token" \
  -d "client_id=backend-client" \
  -d "client_secret=backend-client-secret" \
  -d "refresh_token=<refresh_token>"
```

## API Endpoints

- `POST /device_authorization` - Device authorization endpoint (RFC 8628)
- `GET/POST /device` - User verification endpoint with web interface
- `POST /token` - Token endpoint (supports all grant types)
- `GET /userinfo` - UserInfo endpoint (OIDC)
- `GET /.well-known/oauth-authorization-server` - OAuth2 discovery endpoint
- `GET /.well-known/openid_configuration` - OpenID Connect discovery
- `GET /` - Interactive web interface for testing

## Client Configuration

### Frontend Client (`frontend-client`)

- **Client ID:** `frontend-client`
- **Grant Types:** `authorization_code`, `refresh_token`, `urn:ietf:params:oauth:grant-type:device_code`
- **Scopes:** `openid`, `profile`, `email`, `offline_access`, `api:read`
- **Audience:** `api-service`, `user-service`
- **Use Case:** User-facing applications, CLI tools, IoT devices

### Backend Client (`backend-client`)

- **Client ID:** `backend-client`
- **Client Secret:** `backend-client-secret`
- **Grant Types:** `client_credentials`, `urn:ietf:params:oauth:grant-type:token-exchange`, `refresh_token`
- **Scopes:** `api:read`, `api:write`
- **Audience:** `api-service`
- **Use Case:** Service-to-service communication, long-running processes

## Architecture & Implementation

The implementation includes:

- **Fosite Integration:** Uses Fosite's compose package for OAuth2/OIDC flows
- **In-Memory Storage:** Simple storage for development (replace with persistent storage for production)
- **JWT Tokens:** RS256 signed tokens with RSA key pair
- **Session Management:** Custom session store for device authorization state
- **Web UI:** Interactive HTML interfaces for testing all OAuth2 flows
- **Comprehensive Testing:** Complete test suite covering all flows

## Project Structure

```text
oauth2-server/
├── main.go                    # Server entry point and client registration
├── handlers.go                # HTTP handlers and OAuth2 endpoints
├── token_exchange.go          # Custom grant type implementations
├── session_store.go           # Device authorization storage
├── sample_clients.go          # Interactive web UI for testing
├── config.go                  # Configuration management
├── test_complete_flow.sh      # Comprehensive test suite
├── Makefile                   # Build and development commands
└── README.md                  # This documentation
```

## Development

### Available Make Targets

```bash
make help           # Show all available commands
make build          # Build the OAuth2 server
make run            # Start the server with helpful output
make test           # Run comprehensive OAuth2 flow tests
make demo           # Start server and open web interface
make dev            # Development build (fmt + vet + build)
make clients        # Show registered client information
make well-known     # Display OAuth2 configuration
make clean          # Clean build artifacts
```

### Testing

Run the comprehensive test suite:

```bash
make test
```

This tests:

- ✅ Authorization Code Flow
- ✅ Client Credentials Flow  
- ✅ Refresh Token Flow
- ✅ Device Code Flow (RFC 8628)
- ✅ Token Exchange (RFC 8693)
- ✅ UserInfo endpoint
- ✅ Well-known configuration

## Test Credentials

- **Username:** `john.doe`
- **Password:** `password123`

## Security Considerations

This is a **development example**. For production use:

- ✅ Use persistent, secure storage (database)
- ✅ Implement proper user authentication and authorization
- ✅ Use HTTPS/TLS with proper certificates
- ✅ Implement rate limiting and DDoS protection
- ✅ Add comprehensive logging and monitoring
- ✅ Use secure secret management (Azure Key Vault, HashiCorp Vault, etc.)
- ✅ Validate all inputs thoroughly
- ✅ Implement proper session management
- ✅ Add audit logging for compliance
- ✅ Use proper JWT libraries with security best practices

## Production Deployment

### Environment Variables

```bash
export OAUTH2_PORT=8080
export OAUTH2_BASE_URL=https://your-domain.com
export OAUTH2_JWT_PRIVATE_KEY_PATH=/path/to/private-key.pem
export OAUTH2_DATABASE_URL=postgresql://user:pass@host/db
```

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Run in production
docker run -p 8080:8080 \
  -e OAUTH2_BASE_URL=https://your-domain.com \
  -e OAUTH2_DATABASE_URL=postgresql://... \
  oauth2-server
```

## Use Cases & Examples

### 1. CLI Application Authentication
Perfect for CLI tools that need to authenticate users without a browser:

```bash
# Your CLI app initiates device flow
my-cli-tool login
# User visits URL and authorizes
# CLI receives tokens and can make API calls
```

### 2. IoT Device Authentication
For devices with limited input capabilities:

```bash
# Device displays user code on screen
# User authorizes on their phone/computer
# Device receives tokens for API access
```

### 3. Service-to-Service Communication
Backend services can exchange tokens for secure API access:

```bash
# Service A gets token via client credentials
# Service A exchanges token for Service B specific token
# Service A calls Service B APIs with exchanged token
```

### 4. Long-Running Processes
Applications that need to maintain access over extended periods:

```bash
# Process gets initial token + refresh token
# Process refreshes token before expiration
# Continuous operation without user intervention
```

## Customization

You can modify:

- **Client configurations** in `main.go` and `config.go`
- **Token lifespans** in the Fosite configuration
- **Scopes and audiences** for your specific use case
- **Storage implementation** for production persistence
- **Authentication mechanism** for your user directory
- **UI/UX** of the verification pages in `sample_clients.go`
- **Well-known configuration** in `handlers.go`

## Troubleshooting

### Common Issues

1. **"authorization_pending" error**
   - User hasn't completed authorization yet
   - Check that user visited verification URL and logged in

2. **"expired_token" error**
   - Device code or user code has expired (default: 10 minutes)
   - Restart device authorization flow

3. **"invalid_client" error**
   - Check client ID and secret are correct
   - Verify client exists in registration

4. **"invalid_grant" error**
   - Token may be expired or invalid
   - Check token format and expiration

5. **Server not responding**
   - Ensure server is running: `make run`
   - Check port 8080 is not in use by another process

### Debug Mode

Enable detailed logging:

```bash
# Set environment variable for debug output
export FOSITE_DEBUG=true
make run
```

### Health Check

```bash
# Check server status
curl -s http://localhost:8080/.well-known/oauth-authorization-server | jq .

# Verify all endpoints
make well-known
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Run tests: `make test`
5. Commit your changes: `git commit -am 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## References & Standards

- 📚 [RFC 8628: OAuth 2.0 Device Authorization Grant](https://tools.ietf.org/html/rfc8628)
- 📚 [RFC 8693: OAuth 2.0 Token Exchange](https://tools.ietf.org/html/rfc8693)
- 📚 [RFC 6749: OAuth 2.0 Authorization Framework](https://tools.ietf.org/html/rfc6749)
- 📚 [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- 🛠️ [Fosite Documentation](https://github.com/ory/fosite)
- 🛠️ [JWT.io](https://jwt.io/) - For JWT debugging
- 🔧 [OAuth 2.0 Playground](https://developers.google.com/oauthplayground) - For testing

---

**Built with ❤️ using [Fosite](https://github.com/ory/fosite) - The security first OAuth2 & OpenID Connect framework for Go.**
