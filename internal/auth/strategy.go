package auth

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"oauth2-server/internal/attestation"
	"oauth2-server/internal/store"
	"oauth2-server/internal/utils"
	"oauth2-server/pkg/config"

	"github.com/ory/fosite"
)

type fositeClientKey string

const proxyTokenContextKey = "proxy_token"

// ClientAuthStrategy implements a composite client authentication strategy
// that handles attestation-based authentication and standard OAuth2 client authentication
type ClientAuthStrategy struct {
	AttestationManager *attestation.VerifierManager
	Storage            store.Storage
	Config             *config.Config
	AssertionVerifier  *AssertionVerifier
}

// NewClientAuthStrategy creates a new composite client authentication strategy
func NewClientAuthStrategy(
	attestationManager *attestation.VerifierManager,
	storage store.Storage,
	cfg *config.Config,
	assertionVerifier *AssertionVerifier,
) fosite.ClientAuthenticationStrategy {
	strategy := &ClientAuthStrategy{
		AttestationManager: attestationManager,
		Storage:            storage,
		Config:             cfg,
		AssertionVerifier:  assertionVerifier,
	}
	return strategy.AuthenticateClient
}

// AuthenticateClient implements the fosite.ClientAuthenticationStrategy interface
// It handles attestation authentication and standard OAuth2 client authentication methods
func (s *ClientAuthStrategy) AuthenticateClient(ctx context.Context, r *http.Request, form url.Values) (fosite.Client, error) {
	// Handle proxy token creation
	if ctx.Value(proxyTokenContextKey) != nil {
		clientID := form.Get("client_id")
		if clientID != "" {
			client, err := s.Storage.GetClient(ctx, clientID)
			if err != nil {
				return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
			}
			// For public clients, wrap to make them appear confidential
			if client.IsPublic() {
				wrappedClient := &PublicClientWrapper{
					Client: client,
				}
				return wrappedClient, nil
			}
			return client, nil
		}
	}

	clientID := form.Get("client_id")

	// Check if this is a proxy token creation request - skip authentication in that case
	if ctx.Value(proxyTokenContextKey) != nil {
		if clientID != "" {
			client, err := s.Storage.GetClient(ctx, clientID)
			if err != nil {
				return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
			}
			// Ensure client_credentials is in grant types for proxy token creation
			grantTypes := client.GetGrantTypes()
			hasClientCredentials := false
			for _, gt := range grantTypes {
				if gt == "client_credentials" {
					hasClientCredentials = true
					break
				}
			}
			if !hasClientCredentials {
				// Create a wrapper client with client_credentials added
				grantWrappedClient := &GrantTypeWrapper{
					Client:          client,
					ExtraGrantTypes: []string{"client_credentials"},
				}
				// For public clients, also make them appear confidential for proxy tokens
				if client.IsPublic() {
					wrappedClient := &PublicClientWrapper{
						Client: grantWrappedClient,
					}
					return wrappedClient, nil
				}
				return grantWrappedClient, nil
			}
			// For public clients, wrap to make them appear confidential
			if client.IsPublic() {
				wrappedClient := &PublicClientWrapper{
					Client: client,
				}
				return wrappedClient, nil
			}
			return client, nil
		}
	}

	// Check if this is a proxy token creation request - return the client from context if available
	if client := ctx.Value(fositeClientKey("client")); client != nil {
		if fositeClient, ok := client.(fosite.Client); ok {
			// For proxy tokens, wrap public clients to make them appear confidential
			if ctx.Value(proxyTokenContextKey) != nil && fositeClient.IsPublic() {
				wrappedClient := &PublicClientWrapper{
					Client: fositeClient,
				}
				return wrappedClient, nil
			}
			return fositeClient, nil
		}
	}

	// First, try attestation authentication if client ID is provided
	if clientID != "" && s.AttestationManager != nil && s.AttestationManager.IsAttestationEnabled(clientID) {
		authMethod := s.determineAuthMethod(form)
		if authMethod != "" {
			client, err := s.authenticateWithAttestation(ctx, form, authMethod)
			if err != nil {
				return nil, err
			}
			if client != nil {
				return client, nil
			}
		}
	}

	// Fall back to standard OAuth2 client authentication methods
	return s.authenticateStandard(ctx, r, form)
}

// GrantTypeWrapper wraps a fosite client to add extra grant types
type GrantTypeWrapper struct {
	fosite.Client
	ExtraGrantTypes []string
}

// GetGrantTypes returns the client's grant types plus any extra ones
func (w *GrantTypeWrapper) GetGrantTypes() fosite.Arguments {
	original := w.Client.GetGrantTypes()
	return append(original, w.ExtraGrantTypes...)
}

// PublicClientWrapper wraps a fosite client to override IsPublic for proxy token creation
type PublicClientWrapper struct {
	fosite.Client
}

// IsPublic returns false for proxy token creation to allow client_credentials grant
func (w *PublicClientWrapper) IsPublic() bool {
	return false
}

// GetHashedSecret returns a fake secret for proxy token creation
func (w *PublicClientWrapper) GetHashedSecret() []byte {
	return []byte("$2a$10$fakehashforproxyclient")
}

// GetTokenEndpointAuthMethod returns client_secret_basic for proxy token creation
func (w *PublicClientWrapper) GetTokenEndpointAuthMethod() string {
	return "client_secret_basic"
}

// AttestationClientWrapper wraps a fosite client to override IsPublic for attestation-authenticated clients
type AttestationClientWrapper struct {
	fosite.Client
	attestationAuthenticated bool
}

// IsPublic returns false for attestation-authenticated clients to allow client_credentials grant
func (w *AttestationClientWrapper) IsPublic() bool {
	if w.attestationAuthenticated {
		return false
	}
	return w.Client.IsPublic()
}

// authenticateWithAttestation handles attestation-based client authentication
func (s *ClientAuthStrategy) authenticateWithAttestation(ctx context.Context, form url.Values, authMethod string) (fosite.Client, error) {
	clientID := form.Get("client_id")

	// Get the appropriate verifier
	verifier, err := s.AttestationManager.GetVerifier(clientID, authMethod)
	if err != nil {
		return nil, err
	}

	// Perform attestation verification based on method
	var result *attestation.AttestationResult

	switch authMethod {
	case "attest_jwt_client_auth":
		// Extract JWT from client_assertion parameter
		clientAssertion := form.Get("client_assertion")
		if clientAssertion == "" {
			return nil, fosite.ErrInvalidRequest.WithHint("Missing client_assertion for JWT attestation")
		}

		if jwtVerifier, ok := verifier.(attestation.AttestationVerifier); ok {
			result, err = jwtVerifier.VerifyAttestation(clientAssertion)
		} else {
			return nil, fosite.ErrServerError.WithHint("Invalid JWT verifier")
		}

	case "attest_tls_client_auth":
		// For TLS attestation, we would need the TLS connection state
		// This would require access to the http.Request
		// For now, we'll skip TLS attestation in this strategy
		return nil, fosite.ErrInvalidRequest.WithHint("TLS attestation not supported")

	default:
		return nil, fosite.ErrInvalidRequest.WithHintf("Unsupported attestation method: %s", authMethod)
	}

	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Attestation verification failed")
	}

	if !result.Valid {
		return nil, fosite.ErrInvalidClient.WithHint("Invalid attestation")
	}

	// Enforce that attestation audience includes this authorization server when configured.
	if s.AssertionVerifier != nil && len(s.AssertionVerifier.AcceptedAudiences) > 0 {
		if !attestationAudienceAccepted(result.Audiences, s.AssertionVerifier.AcceptedAudiences) {
			return nil, fosite.ErrInvalidClient.WithHint("Attestation audience does not match this authorization server")
		}
	}

	// Get the client from store
	client, err := s.Storage.GetClient(ctx, result.ClientID)
	if err != nil {
		return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
	}

	// Store attestation result in request context for later use
	attestation.WithAttestationResult(ctx, result)

	// Wrap the client to override IsPublic for attestation authentication
	wrappedClient := &AttestationClientWrapper{
		Client:                   client,
		attestationAuthenticated: true,
	}

	return wrappedClient, nil
}

func attestationAudienceAccepted(got, accepted []string) bool {
	for _, want := range accepted {
		if want == "" {
			continue
		}
		for _, a := range got {
			if a == want {
				return true
			}
		}
	}
	return false
}

// authenticateStandard handles standard OAuth2 client authentication methods
func (s *ClientAuthStrategy) authenticateStandard(ctx context.Context, r *http.Request, form url.Values) (fosite.Client, error) {
	clientID := form.Get("client_id")

	// Method 1: HTTP Basic Authentication
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Basic ") {
			if decoded, err := base64.StdEncoding.DecodeString(auth[6:]); err == nil {
				if parts := strings.SplitN(string(decoded), ":", 2); len(parts) == 2 {
					reqClientID, clientSecret := parts[0], parts[1]

					// If client_id was provided in form, it must match the one in Basic auth
					if clientID != "" && clientID != reqClientID {
						return nil, fosite.ErrInvalidClient.WithHint("Client ID mismatch between form and authorization header")
					}

					clientID = reqClientID

					client, err := s.Storage.GetClient(ctx, clientID)
					if err != nil {
						return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
					}

					// Verify client secret using proper bcrypt validation
					if !utils.ValidateSecret(clientSecret, client.GetHashedSecret()) {
						return nil, fosite.ErrInvalidClient.WithHint("Invalid client credentials")
					}

					return client, nil
				}
			}
		}
	}

	// Method 2: client_secret_post
	if clientID != "" {
		clientSecret := form.Get("client_secret")
		if clientSecret != "" {
			client, err := s.Storage.GetClient(ctx, clientID)
			if err != nil {
				return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
			}

			// Verify client secret using proper bcrypt validation
			if !utils.ValidateSecret(clientSecret, client.GetHashedSecret()) {
				return nil, fosite.ErrInvalidClient.WithHint("Invalid client credentials")
			}

			return client, nil
		}
	}

	// Method 3: RFC 7523 private_key_jwt / client_secret_jwt
	clientAssertionType := form.Get("client_assertion_type")
	clientAssertion := form.Get("client_assertion")
	if clientAssertionType == clientAssertionJWTBearerType && clientAssertion != "" {
		if s.AssertionVerifier == nil {
			return nil, fosite.ErrServerError.WithHint("JWT client assertion authentication is not configured")
		}
		return s.AssertionVerifier.Authenticate(ctx, clientAssertionType, clientAssertion, clientID)
	}

	// Method 4: Public client (no authentication required)
	if clientID != "" {
		client, err := s.Storage.GetClient(ctx, clientID)
		if err != nil {
			return nil, fosite.ErrInvalidClient.WithHint("Unknown client")
		}

		// Check if client is public (no secret required)
		if client.IsPublic() || len(client.GetHashedSecret()) == 0 {
			return client, nil
		}
	}

	return nil, fosite.ErrInvalidClient.WithHint("Client authentication failed")
}

// determineAuthMethod determines the attestation authentication method from the form
func (s *ClientAuthStrategy) determineAuthMethod(form url.Values) string {
	clientID := form.Get("client_id")

	// Check for JWT attestation
	clientAssertionType := form.Get("client_assertion_type")
	if clientAssertionType == clientAssertionJWTBearerType {
		clientAssertion := form.Get("client_assertion")

		// Check if the client is configured for attestation-based authentication
		if s.AttestationManager != nil && s.AttestationManager.IsAttestationEnabled(clientID) {
			supportedMethods, err := s.AttestationManager.GetSupportedMethods(clientID)
			if err == nil {
				// Check if attest_jwt_client_auth is supported
				for _, method := range supportedMethods {
					if method == "attest_jwt_client_auth" {
						return "attest_jwt_client_auth"
					}
				}
			}
		}

		// Also check if JWT contains attestation-specific claims
		if strings.Contains(clientAssertion, "att_") {
			return "attest_jwt_client_auth"
		}
	}

	return ""
}
