# Fosite OAuth2 Example

This project demonstrates an OpenID Connect provider implementation using the [Fosite](https://github.com/ory/fosite) library with support for:

- **Device Code Flow (RFC 8628)** - For devices with limited input capabilities
- **Token Exchange (RFC 8693)** - For service-to-service token exchange
- **Refresh Tokens** - For long-running applications
- **Two Client Setup** - Device client and service client with audience-based tokens

## Features

### Client 1: Device Code Flow
- Initiates device authorization flow
- Supports OpenID Connect scopes
- Issues tokens with audience scope for Client 2
- Provides refresh tokens for token renewal

### Client 2: Token Exchange
- Exchanges tokens from Client 1 for service-specific tokens
- Supports RFC 8693 token exchange
- Long-running process with refresh token capability
- Audience validation

## Prerequisites

- Go 1.21 or later
- No external dependencies required (uses in-memory storage)

## Installation

1. Clone or download the project
2. Install dependencies:
```bash
go mod tidy
```

## Running the Server

```bash
go run .
```

The server will start on `http://localhost:8080`

## Usage

### 1. Start the Server
```bash
go run .
```

### 2. Test Device Code Flow (Client 1)

Visit `http://localhost:8080/client1/device` or use curl:

```bash
# Initiate device authorization
curl -X POST http://localhost:8080/device \
  -d "client_id=device-client" \
  -d "scope=openid profile email offline_access"
```

This returns:
```json
{
  "device_code": "...",
  "user_code": "ABCD-EFGH",
  "verification_uri": "http://localhost:8080/device/auth",
  "verification_uri_complete": "http://localhost:8080/device/auth?user_code=ABCD-EFGH",
  "expires_in": 600,
  "interval": 5
}
```

### 3. User Authorization

1. Visit the `verification_uri_complete` URL
2. Enter credentials:
   - **Username:** `john.doe`
   - **Password:** `password123`

### 4. Poll for Token

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:device_code" \
  -d "device_code=<device_code>" \
  -d "client_id=device-client"
```

### 5. Token Exchange (Client 2)

Visit `http://localhost:8080/client2/exchange` or use curl:

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=urn:ietf:params:oauth:grant-type:token-exchange" \
  -d "client_id=service-client" \
  -d "client_secret=service-client-secret" \
  -d "subject_token=<access_token_from_client1>" \
  -d "subject_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "requested_token_type=urn:ietf:params:oauth:token-type:access_token" \
  -d "audience=api-service"
```

### 6. Refresh Tokens

```bash
curl -X POST http://localhost:8080/token \
  -d "grant_type=refresh_token" \
  -d "client_id=service-client" \
  -d "client_secret=service-client-secret" \
  -d "refresh_token=<refresh_token>"
```

## API Endpoints

- `POST /device` - Device authorization endpoint
- `GET/POST /device/auth` - User verification endpoint
- `POST /token` - Token endpoint (supports multiple grant types)
- `GET /userinfo` - UserInfo endpoint
- `GET /.well-known/openid_configuration` - OpenID Connect discovery

## Client Configuration

### Client 1 (Device Client)
- **Client ID:** `device-client`
- **Grant Types:** `urn:ietf:params:oauth:grant-type:device_code`, `refresh_token`
- **Scopes:** `openid`, `profile`, `email`, `offline_access`
- **Audience:** `api-service`, `user-service`

### Client 2 (Service Client)
- **Client ID:** `service-client`
- **Client Secret:** `service-client-secret`
- **Grant Types:** `urn:ietf:params:oauth:grant-type:token-exchange`, `refresh_token`
- **Scopes:** `api:read`, `api:write`
- **Audience:** `api-service`

## Test Credentials

- **Username:** `john.doe`
- **Password:** `password123`

## Architecture

The implementation includes:

- **Fosite Integration:** Uses Fosite's compose package for OAuth2/OIDC flows
- **In-Memory Storage:** Simple storage for development (replace with persistent storage for production)
- **JWT Tokens:** RS256 signed tokens with RSA key pair
- **Session Management:** Custom session store for device authorization state
- **Web UI:** Simple HTML interfaces for testing flows

## Security Considerations

This is a **development example**. For production use:

- Use persistent, secure storage (database)
- Implement proper user authentication
- Use HTTPS/TLS
- Implement rate limiting
- Add proper logging and monitoring
- Use secure secret management
- Validate all inputs thoroughly

## Customization

You can modify:

- Client configurations in `main.go`
- Token lifespans in the config
- Scopes and audiences
- Storage implementation
- Authentication mechanism
- UI/UX of the verification page

## Troubleshooting

### Common Issues

1. **"authorization_pending" error:** User hasn't completed authorization yet
2. **"expired_token" error:** Device code or user code has expired
3. **"invalid_client" error:** Check client credentials
4. **CORS issues:** Add appropriate CORS headers if accessing from browser

### Debug Mode

Add debug logging by modifying the Fosite configuration:

```go
config.EnableDebugMode = true
```

## References

- [RFC 8628: OAuth 2.0 Device Authorization Grant](https://tools.ietf.org/html/rfc8628)
- [RFC 8693: OAuth 2.0 Token Exchange](https://tools.ietf.org/html/rfc8693)
- [Fosite Documentation](https://github.com/ory/fosite)
- [OpenID Connect Specification](https://openid.net/connect/)
