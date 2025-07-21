# OAuth2 Server

This project implements an OAuth2 server that supports various authorization flows, including client credentials, authorization code, device code, and token exchange. It is designed to allow clients to authenticate and obtain access tokens securely.

## Project Structure

- **cmd/server/main.go**: Entry point of the application. Initializes the server and sets up routes and middleware.
- **internal/auth/**: Contains functions and types related to client authentication and token validation.
  - `client_auth.go`: Functions for authenticating clients based on their credentials.
  - `token_validation.go`: Functions for validating tokens, ensuring they are correctly signed and not expired.
- **internal/flows/**: Implements various OAuth2 flows.
  - `authorization_code.go`: Handles the authorization code flow.
  - `client_credentials.go`: Implements the client credentials flow.
  - `device_code.go`: Manages the device code flow.
  - `refresh_token.go`: Processes refresh token requests.
  - `token_exchange.go`: Handles token exchange requests.
- **internal/registration/**: Functions and types related to dynamic client registration.
  - `dynamic_client_registration.go`: Allows clients to register themselves with the server.
- **internal/handlers/**: Defines HTTP handlers for various endpoints.
  - `auth_handlers.go`: Handlers for authentication-related endpoints.
  - `device_handlers.go`: Handlers for device authorization endpoints.
  - `registration_handlers.go`: Handlers for client registration endpoints.
  - `token_handlers.go`: Handlers for token-related endpoints.
- **internal/models/**: Defines data models used in the application.
  - `client.go`: Represents registered clients.
  - `device.go`: Represents devices that can be authorized.
  - `requests.go`: Structures for various request types.
  - `responses.go`: Structures for various response types.
- **internal/store/**: Manages storage and retrieval of data.
  - `client_store.go`: Storage for client data.
  - `device_store.go`: Storage for device data.
  - `token_store.go`: Storage for tokens.
- **internal/utils/**: Utility functions used throughout the application.
  - `generators.go`: Functions for generating random strings and tokens.
  - `helpers.go`: Various helper functions.
  - `validators.go`: Validation functions for requests and tokens.
- **pkg/config/**: Configuration management for the application.
  - `config.go`: Loads settings from environment variables or configuration files.
- **go.mod**: Defines the module and its dependencies.
- **go.sum**: Contains checksums for the module's dependencies.

## Setup Instructions

1. Clone the repository:
   ```
   git clone <repository-url>
   cd oauth2-server
   ```

2. Install dependencies:
   ```
   go mod tidy
   ```

3. Run the server:
   ```
   go run cmd/server/main.go
   ```

## Usage Guidelines

- The server supports various OAuth2 flows. Refer to the specific flow documentation for details on how to use each endpoint.
- Ensure that you have the necessary client credentials for authentication.
- Use tools like Postman or curl to test the endpoints.

## Contributing

Contributions are welcome! Please submit a pull request or open an issue for any enhancements or bug fixes.