package registration

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"oauth2-server/internal/store"
	"oauth2-server/pkg/config"
)

// Define complete ClientRegistrationRequest struct
type ClientRegistrationRequest struct {
	ClientName                   string            `json:"client_name,omitempty"`
	ClientURI                    string            `json:"client_uri,omitempty"`
	LogoURI                      string            `json:"logo_uri,omitempty"`
	Scope                        string            `json:"scope,omitempty"`
	Contacts                     []string          `json:"contacts,omitempty"`
	TosURI                       string            `json:"tos_uri,omitempty"`
	PolicyURI                    string            `json:"policy_uri,omitempty"`
	JwksURI                      string            `json:"jwks_uri,omitempty"`
	Jwks                         interface{}       `json:"jwks,omitempty"`
	SoftwareID                   string            `json:"software_id,omitempty"`
	SoftwareVersion              string            `json:"software_version,omitempty"`
	RedirectURIs                 []string          `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod      string            `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes                   []string          `json:"grant_types,omitempty"`
	ResponseTypes                []string          `json:"response_types,omitempty"`
	ApplicationType              string            `json:"application_type,omitempty"`
	SectorIdentifierURI          string            `json:"sector_identifier_uri,omitempty"`
	SubjectType                  string            `json:"subject_type,omitempty"`
	RequestObjectSigningAlg      string            `json:"request_object_signing_alg,omitempty"`
	UserinfoSignedResponseAlg    string            `json:"userinfo_signed_response_alg,omitempty"`
	UserinfoEncryptedResponseAlg string            `json:"userinfo_encrypted_response_alg,omitempty"`
	UserinfoEncryptedResponseEnc string            `json:"userinfo_encrypted_response_enc,omitempty"`
	IDTokenSignedResponseAlg     string            `json:"id_token_signed_response_alg,omitempty"`
	IDTokenEncryptedResponseAlg  string            `json:"id_token_encrypted_response_alg,omitempty"`
	IDTokenEncryptedResponseEnc  string            `json:"id_token_encrypted_response_enc,omitempty"`
	DefaultMaxAge                *int              `json:"default_max_age,omitempty"`
	RequireAuthTime              *bool             `json:"require_auth_time,omitempty"`
	DefaultACRValues             []string          `json:"default_acr_values,omitempty"`
	InitiateLoginURI             string            `json:"initiate_login_uri,omitempty"`
	RequestURIs                  []string          `json:"request_uris,omitempty"`
}

// Define complete ClientRegistrationResponse struct
type ClientRegistrationResponse struct {
	ClientID                     string            `json:"client_id"`
	ClientSecret                 string            `json:"client_secret,omitempty"`
	ClientSecretExpiresAt        int64             `json:"client_secret_expires_at"`
	RegistrationAccessToken      string            `json:"registration_access_token"`
	RegistrationClientURI        string            `json:"registration_client_uri"`
	ClientName                   string            `json:"client_name,omitempty"`
	ClientURI                    string            `json:"client_uri,omitempty"`
	LogoURI                      string            `json:"logo_uri,omitempty"`
	Scope                        string            `json:"scope,omitempty"`
	Contacts                     []string          `json:"contacts,omitempty"`
	TosURI                       string            `json:"tos_uri,omitempty"`
	PolicyURI                    string            `json:"policy_uri,omitempty"`
	JwksURI                      string            `json:"jwks_uri,omitempty"`
	Jwks                         interface{}       `json:"jwks,omitempty"`
	SoftwareID                   string            `json:"software_id,omitempty"`
	SoftwareVersion              string            `json:"software_version,omitempty"`
	RedirectURIs                 []string          `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod      string            `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes                   []string          `json:"grant_types,omitempty"`
	ResponseTypes                []string          `json:"response_types,omitempty"`
	ApplicationType              string            `json:"application_type,omitempty"`
	SectorIdentifierURI          string            `json:"sector_identifier_uri,omitempty"`
	SubjectType                  string            `json:"subject_type,omitempty"`
	RequestObjectSigningAlg      string            `json:"request_object_signing_alg,omitempty"`
	UserinfoSignedResponseAlg    string            `json:"userinfo_signed_response_alg,omitempty"`
	UserinfoEncryptedResponseAlg string            `json:"userinfo_encrypted_response_alg,omitempty"`
	UserinfoEncryptedResponseEnc string            `json:"userinfo_encrypted_response_enc,omitempty"`
	IDTokenSignedResponseAlg     string            `json:"id_token_signed_response_alg,omitempty"`
	IDTokenEncryptedResponseAlg  string            `json:"id_token_encrypted_response_alg,omitempty"`
	IDTokenEncryptedResponseEnc  string            `json:"id_token_encrypted_response_enc,omitempty"`
	DefaultMaxAge                *int              `json:"default_max_age,omitempty"`
	RequireAuthTime              *bool             `json:"require_auth_time,omitempty"`
	DefaultACRValues             []string          `json:"default_acr_values,omitempty"`
	InitiateLoginURI             string            `json:"initiate_login_uri,omitempty"`
	RequestURIs                  []string          `json:"request_uris,omitempty"`
	CreatedAt                    time.Time         `json:"created_at"`
	UpdatedAt                    time.Time         `json:"updated_at"`
}

// Define complete RegisteredClient struct
type RegisteredClient struct {
	ID                         string
	Secret                     string
	SecretExpiresAt            int64
	RegistrationAccessToken    string
	RedirectURIs               []string
	TokenEndpointAuthMethod    string
	GrantTypes                 []string
	ResponseTypes              []string
	Name                       string
	URI                        string
	LogoURI                    string
	Scope                      string
	Contacts                   []string
	TosURI                     string
	PolicyURI                  string
	JwksURI                    string
	Jwks                       interface{}
	SoftwareID                 string
	SoftwareVersion            string
	ApplicationType            string
	SectorIdentifierURI        string
	SubjectType                string
	RequestObjectSigningAlg    string
	UserinfoSignedResponseAlg  string
	IDTokenSignedResponseAlg   string
	DefaultMaxAge              *int
	RequireAuthTime            *bool
	DefaultACRValues           []string
	InitiateLoginURI           string
	RequestURIs                []string
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// RegisterClient processes a client registration request and returns a registered client
func RegisterClient(request *ClientRegistrationRequest, clientStore *store.ClientStore, config *config.Config) (*ClientRegistrationResponse, error) {
	// Generate client ID
	clientID, err := generateClientID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate client ID: %w", err)
	}

	// Generate client secret if required
	var clientSecret string
	var secretExpiresAt int64

	authMethod := request.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_basic" // Default
	}

	if authMethod != "none" {
		clientSecret, err = generateClientSecret()
		if err != nil {
			return nil, fmt.Errorf("failed to generate client secret: %w", err)
		}
		// Set expiration to 1 year from now (0 means never expires)
		secretExpiresAt = time.Now().Add(365 * 24 * time.Hour).Unix()
	}

	// Set default values if not provided
	grantTypes := request.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code"}
	}

	responseTypes := request.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	redirectURIs := request.RedirectURIs
	if len(redirectURIs) == 0 && contains(grantTypes, "authorization_code") {
		return nil, fmt.Errorf("redirect_uris is required for authorization_code grant")
	}

	applicationType := request.ApplicationType
	if applicationType == "" {
		applicationType = "web"
	}

	subjectType := request.SubjectType
	if subjectType == "" {
		subjectType = "public"
	}

	scope := request.Scope
	if scope == "" {
		scope = "openid profile email"
	}

	// Generate registration access token
	registrationAccessToken, err := generateRegistrationAccessToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate registration access token: %w", err)
	}

	// Create the registered client
	registeredClient := &RegisteredClient{
		ID:                         clientID,
		Secret:                     clientSecret,
		SecretExpiresAt:            secretExpiresAt,
		RegistrationAccessToken:    registrationAccessToken,
		RedirectURIs:               redirectURIs,
		TokenEndpointAuthMethod:    authMethod,
		GrantTypes:                 grantTypes,
		ResponseTypes:              responseTypes,
		Name:                       request.ClientName,
		URI:                        request.ClientURI,
		LogoURI:                    request.LogoURI,
		Scope:                      scope,
		Contacts:                   request.Contacts,
		TosURI:                     request.TosURI,
		PolicyURI:                  request.PolicyURI,
		JwksURI:                    request.JwksURI,
		Jwks:                       request.Jwks,
		SoftwareID:                 request.SoftwareID,
		SoftwareVersion:            request.SoftwareVersion,
		ApplicationType:            applicationType,
		SectorIdentifierURI:        request.SectorIdentifierURI,
		SubjectType:                subjectType,
		RequestObjectSigningAlg:    request.RequestObjectSigningAlg,
		UserinfoSignedResponseAlg:  request.UserinfoSignedResponseAlg,
		IDTokenSignedResponseAlg:   request.IDTokenSignedResponseAlg,
		DefaultMaxAge:              request.DefaultMaxAge,
		RequireAuthTime:            request.RequireAuthTime,
		DefaultACRValues:           request.DefaultACRValues,
		InitiateLoginURI:           request.InitiateLoginURI,
		RequestURIs:                request.RequestURIs,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	// Store the client
	storeClient := &store.Client{
		ID:           clientID,
		Secret:       []byte(clientSecret), // Fix: convert string to []byte
		RedirectURIs: redirectURIs,
		GrantTypes:   grantTypes,
		ResponseTypes: responseTypes,
		Scopes:       strings.Split(scope, " "),
		Name:         request.ClientName,
		Public:       authMethod == "none",
	}

	if err := clientStore.StoreClient(storeClient); err != nil {
		return nil, fmt.Errorf("failed to store client: %w", err)
	}

	// Create response
	response := &ClientRegistrationResponse{
		ClientID:                    clientID,
		ClientSecret:                clientSecret,
		ClientSecretExpiresAt:       secretExpiresAt,
		RegistrationAccessToken:     registrationAccessToken,
		RegistrationClientURI:       fmt.Sprintf("%s/register/%s", config.BaseURL, clientID),
		RedirectURIs:                redirectURIs,
		TokenEndpointAuthMethod:     authMethod,
		GrantTypes:                  grantTypes,
		ResponseTypes:               responseTypes,
		ClientName:                  request.ClientName,
		ClientURI:                   request.ClientURI,
		LogoURI:                     request.LogoURI,
		Scope:                       scope,
		Contacts:                    request.Contacts,
		TosURI:                      request.TosURI,
		PolicyURI:                   request.PolicyURI,
		JwksURI:                     request.JwksURI,
		Jwks:                        request.Jwks,
		SoftwareID:                  request.SoftwareID,
		SoftwareVersion:             request.SoftwareVersion,
		ApplicationType:             applicationType,
		SectorIdentifierURI:         request.SectorIdentifierURI,
		SubjectType:                 subjectType,
		RequestObjectSigningAlg:     request.RequestObjectSigningAlg,
		UserinfoSignedResponseAlg:   request.UserinfoSignedResponseAlg,
		UserinfoEncryptedResponseAlg: request.UserinfoEncryptedResponseAlg,
		UserinfoEncryptedResponseEnc: request.UserinfoEncryptedResponseEnc,
		IDTokenSignedResponseAlg:    request.IDTokenSignedResponseAlg,
		IDTokenEncryptedResponseAlg: request.IDTokenEncryptedResponseAlg,
		IDTokenEncryptedResponseEnc: request.IDTokenEncryptedResponseEnc,
		DefaultMaxAge:               request.DefaultMaxAge,
		RequireAuthTime:             request.RequireAuthTime,
		DefaultACRValues:            request.DefaultACRValues,
		InitiateLoginURI:            request.InitiateLoginURI,
		RequestURIs:                 request.RequestURIs,
		CreatedAt:                   registeredClient.CreatedAt,
		UpdatedAt:                   registeredClient.UpdatedAt,
	}

	return response, nil
}

// Helper functions
func generateClientID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "client_" + hex.EncodeToString(bytes), nil
}

func generateClientSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "secret_" + hex.EncodeToString(bytes), nil
}

func generateRegistrationAccessToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "reg_" + hex.EncodeToString(bytes), nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestDynamicClientRegistration(t *testing.T) {
	// Create a mock client store
	clientStore := store.NewClientStore()

	// Create a mock config
	cfg := &config.Config{
		BaseURL: "http://localhost:8080",
	}

	// Mock request
	request := &ClientRegistrationRequest{
		ClientName:    "Test Client",
		RedirectURIs:  []string{"http://localhost:8080/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
		Scope:         "openid profile",
	}

	t.Run("successful_registration", func(t *testing.T) {
		// Test client registration
		response, err := RegisterClient(request, clientStore, cfg)
		if err != nil {
			t.Fatalf("RegisterClient() error = %v", err)
		}

		// Test assertions
		if response.ClientID == "" {
			t.Error("ClientID should not be empty")
		}

		if response.ClientSecret == "" {
			t.Error("ClientSecret should not be empty")
		}

		if response.ClientName != request.ClientName {
			t.Errorf("Expected ClientName %s, got %s", request.ClientName, response.ClientName)
		}

		if len(response.RedirectURIs) != len(request.RedirectURIs) {
			t.Error("RedirectURIs length mismatch")
		}

		if len(response.GrantTypes) != len(request.GrantTypes) {
			t.Error("GrantTypes length mismatch")
		}
	})

	t.Run("registration_with_defaults", func(t *testing.T) {
		// Test with minimal request
		minimalRequest := &ClientRegistrationRequest{
			ClientName:   "Minimal Client",
			RedirectURIs: []string{"http://localhost:8080/callback"},
		}

		response, err := RegisterClient(minimalRequest, clientStore, cfg)
		if err != nil {
			t.Fatalf("RegisterClient() error = %v", err)
		}

		// Check defaults were applied
		if len(response.GrantTypes) == 0 {
			t.Error("Default grant types should be applied")
		}

		if len(response.ResponseTypes) == 0 {
			t.Error("Default response types should be applied")
		}

		if response.TokenEndpointAuthMethod == "" {
			t.Error("Default auth method should be applied")
		}
	})

	t.Run("registration_without_required_redirect_uri", func(t *testing.T) {
		// Test with missing redirect URI for authorization_code grant
		invalidRequest := &ClientRegistrationRequest{
			ClientName: "Invalid Client",
			GrantTypes: []string{"authorization_code"},
			// Missing RedirectURIs
		}

		_, err := RegisterClient(invalidRequest, clientStore, cfg)
		if err == nil {
			t.Error("Expected error for missing redirect_uris with authorization_code grant")
		}
	})
}