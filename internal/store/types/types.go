package types

import (
	"context"
	"encoding/json"
	"time"

	"oauth2-server/pkg/config"

	"github.com/go-jose/go-jose/v3"
	"github.com/ory/fosite"
	"github.com/ory/fosite/storage"
)

// CustomClient extends fosite.DefaultClient with claims support and RFC 7523 / OIDC client auth fields.
type CustomClient struct {
	*fosite.DefaultClient
	Claims              []string `json:"claims,omitempty"`
	ForceAuthentication bool     `json:"force_authentication,omitempty"`
	ForceConsent        bool     `json:"force_consent,omitempty"`

	// OpenID Connect / RFC 7523 client authentication fields
	JSONWebKeysURI                    string              `json:"jwks_uri,omitempty"`
	JSONWebKeys                       *jose.JSONWebKeySet `json:"jwks,omitempty"`
	TokenEndpointAuthMethod           string              `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthSigningAlgorithm string              `json:"token_endpoint_auth_signing_alg,omitempty"`
	RequestURIs                       []string            `json:"request_uris,omitempty"`
	RequestObjectSigningAlgorithm     string              `json:"request_object_signing_alg,omitempty"`

	// CIMD fields
	MetadataDocumentLocation     string `json:"metadata_document_location,omitempty"`
	MetadataDocumentExpiresAt    int64  `json:"metadata_document_expires_at,omitempty"`
	MetadataDocumentUpdatedAt    int64  `json:"metadata_document_updated_at,omitempty"`
	DiscoveredByMetadataDocument bool   `json:"discovered_by_metadata_document,omitempty"`
}

// Ensure CustomClient implements fosite.OpenIDConnectClient for private_key_jwt.
var _ fosite.OpenIDConnectClient = (*CustomClient)(nil)

// GetJSONWebKeysURI returns the client's jwks_uri.
func (c *CustomClient) GetJSONWebKeysURI() string {
	return c.JSONWebKeysURI
}

// GetJSONWebKeys returns the client's registered JWKS.
func (c *CustomClient) GetJSONWebKeys() *jose.JSONWebKeySet {
	return c.JSONWebKeys
}

// GetTokenEndpointAuthMethod returns the registered token endpoint auth method.
func (c *CustomClient) GetTokenEndpointAuthMethod() string {
	if c.TokenEndpointAuthMethod != "" {
		return c.TokenEndpointAuthMethod
	}
	if c.IsPublic() {
		return "none"
	}
	return "client_secret_basic"
}

// GetTokenEndpointAuthSigningAlgorithm returns the JWT auth signing algorithm.
func (c *CustomClient) GetTokenEndpointAuthSigningAlgorithm() string {
	if c.TokenEndpointAuthSigningAlgorithm == "" {
		if c.GetTokenEndpointAuthMethod() == "client_secret_jwt" {
			return "HS256"
		}
		return "RS256"
	}
	return c.TokenEndpointAuthSigningAlgorithm
}

// GetRequestURIs returns pre-registered request_uri values.
func (c *CustomClient) GetRequestURIs() []string {
	return c.RequestURIs
}

// GetRequestObjectSigningAlgorithm returns the request object signing algorithm.
func (c *CustomClient) GetRequestObjectSigningAlgorithm() string {
	return c.RequestObjectSigningAlgorithm
}

// PARRequest represents a pushed authorization request
type PARRequest struct {
	RequestURI string            `json:"request_uri"`
	ClientID   string            `json:"client_id"`
	ExpiresAt  time.Time         `json:"expires_at"`
	Parameters map[string]string `json:"parameters"`
}

// GetClaims returns the registered claims for this client
func (c *CustomClient) GetClaims() []string {
	return c.Claims
}

// SetClaims sets the registered claims for this client
func (c *CustomClient) SetClaims(claims []string) {
	c.Claims = claims
}

// RequestWithClientID wraps a fosite.Requester with client ID stored separately for proper JSON marshaling
type RequestWithClientID struct {
	Request       *fosite.Request       `json:"-"`
	AccessRequest *fosite.AccessRequest `json:"-"`
	ClientID      string                `json:"_client_id"`
	Type          string                `json:"_type"`
}

// DeviceRequestWithClientID wraps a fosite.DeviceRequester with client ID stored separately for proper JSON marshaling
type DeviceRequestWithClientID struct {
	DeviceRequest *fosite.DeviceRequest `json:"-"`
	ClientID      string                `json:"_client_id"`
	Type          string                `json:"_type"`
}

// MarshalRequestWithClientID marshals a fosite.Requester with client ID stored separately
func MarshalRequestWithClientID(request fosite.Requester) ([]byte, error) {
	raw := make(map[string]interface{})

	// Get the underlying request data
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(requestData, &raw); err != nil {
		return nil, err
	}

	// Add type information
	raw["_type"] = "Request"
	if client := request.GetClient(); client != nil {
		raw["_client_id"] = client.GetID()
	}

	return json.Marshal(raw)
}

// MarshalDeviceRequestWithClientID marshals a fosite.DeviceRequester with client ID stored separately
func MarshalDeviceRequestWithClientID(request fosite.DeviceRequester) ([]byte, error) {
	raw := make(map[string]interface{})

	// Get the underlying request data
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(requestData, &raw); err != nil {
		return nil, err
	}

	// Add type information
	raw["_type"] = "DeviceRequest"
	if client := request.GetClient(); client != nil {
		raw["_client_id"] = client.GetID()
	}

	return json.Marshal(raw)
}

// Storage interface that both MemoryStore and SQLiteStore implement
type Storage interface {
	// Client storage methods
	GetClient(ctx context.Context, id string) (fosite.Client, error)
	CreateClient(ctx context.Context, client fosite.Client) error
	UpdateClient(ctx context.Context, id string, client fosite.Client) error
	DeleteClient(ctx context.Context, id string) error

	// User storage methods
	GetUser(ctx context.Context, id string) (*storage.MemoryUserRelation, error)

	// Token storage methods
	CreateAccessTokenSession(ctx context.Context, signature string, request fosite.Requester) error
	GetAccessTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error)
	DeleteAccessTokenSession(ctx context.Context, signature string) error
	CreateRefreshTokenSession(ctx context.Context, signature string, accessTokenSignature string, request fosite.Requester) error
	GetRefreshTokenSession(ctx context.Context, signature string, session fosite.Session) (fosite.Requester, error)
	DeleteRefreshTokenSession(ctx context.Context, signature string) error
	RotateRefreshToken(ctx context.Context, requestID string, refreshTokenSignature string) error
	RevokeAccessToken(ctx context.Context, requestID string) error
	RevokeRefreshToken(ctx context.Context, requestID string) error
	CreateAuthorizeCodeSession(ctx context.Context, code string, request fosite.Requester) error
	GetAuthorizeCodeSession(ctx context.Context, code string, session fosite.Session) (fosite.Requester, error)
	InvalidateAuthorizeCodeSession(ctx context.Context, code string) error

	// PKCE methods
	CreatePKCERequestSession(ctx context.Context, code string, request fosite.Requester) error
	GetPKCERequestSession(ctx context.Context, code string, session fosite.Session) (fosite.Requester, error)
	DeletePKCERequestSession(ctx context.Context, code string) error

	// Client Assertion JWT methods
	ClientAssertionJWTValid(ctx context.Context, jti string) error
	SetClientAssertionJWT(ctx context.Context, jti string, exp time.Time) error

	// Device authorization methods
	GetDeviceCodeSession(ctx context.Context, deviceCode string, session fosite.Session) (fosite.DeviceRequester, error)
	CreateDeviceCodeSession(ctx context.Context, deviceCode string, request fosite.Requester) error
	UpdateDeviceCodeSession(ctx context.Context, deviceCode string, request fosite.Requester) error
	InvalidateDeviceCodeSession(ctx context.Context, signature string) error
	GetPendingDeviceAuths(ctx context.Context) (map[string]fosite.Requester, error)
	GetDeviceAuthByUserCode(ctx context.Context, userCode string) (fosite.DeviceRequester, string, error)
	CreateDeviceAuthSession(ctx context.Context, deviceCodeSignature, userCodeSignature string, request fosite.DeviceRequester) error

	// Statistics methods
	GetClientCount() (int, error)
	GetUserCount() (int, error)
	GetAccessTokenCount() (int, error)
	GetRefreshTokenCount() (int, error)

	// Secure client data storage methods
	StoreClientSecret(ctx context.Context, clientID string, encryptedSecret string) error
	GetClientSecret(ctx context.Context, clientID string) (string, error)
	DeleteClientSecret(ctx context.Context, clientID string) error
	StoreAttestationConfig(ctx context.Context, clientID string, config *config.ClientAttestationConfig) error
	GetAttestationConfig(ctx context.Context, clientID string) (*config.ClientAttestationConfig, error)
	DeleteAttestationConfig(ctx context.Context, clientID string) error

	// Trust anchor storage methods
	StoreTrustAnchor(ctx context.Context, name string, certificateData []byte) error
	GetTrustAnchor(ctx context.Context, name string) ([]byte, error)
	ListTrustAnchors(ctx context.Context) ([]string, error)
	DeleteTrustAnchor(ctx context.Context, name string) error

	// Upstream token mapping methods for proxy mode
	StoreUpstreamTokenMapping(ctx context.Context, proxyTokenSignature string, upstreamAccessToken string, upstreamRefreshToken string, upstreamTokenType string, upstreamExpiresIn int64) error
	GetUpstreamTokenMapping(ctx context.Context, proxyTokenSignature string) (upstreamAccessToken string, upstreamRefreshToken string, upstreamTokenType string, upstreamExpiresIn int64, err error)
	DeleteUpstreamTokenMapping(ctx context.Context, proxyTokenSignature string) error

	// PAR (Pushed Authorization Request) methods
	StorePARRequest(ctx context.Context, request *PARRequest) error
	GetPARRequest(ctx context.Context, requestURI string) (*PARRequest, error)
	DeletePARRequest(ctx context.Context, requestURI string) error
}
