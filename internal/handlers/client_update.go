package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"oauth2-server/internal/auth"
	"oauth2-server/internal/store/types"
	"oauth2-server/internal/utils"
	"oauth2-server/pkg/config"

	"github.com/ory/fosite"
)

// ClientPatchMetadata supports selective updates. Pointer fields are applied only when present.
type ClientPatchMetadata struct {
	ClientSecret                      *string                         `json:"client_secret,omitempty"`
	RedirectURIs                      *[]string                       `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod           *string                         `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthSigningAlgorithm *string                         `json:"token_endpoint_auth_signing_alg,omitempty"`
	GrantTypes                        *[]string                       `json:"grant_types,omitempty"`
	ResponseTypes                     *[]string                       `json:"response_types,omitempty"`
	ClientName                        *string                         `json:"client_name,omitempty"`
	ClientURI                         *string                         `json:"client_uri,omitempty"`
	LogoURI                           *string                         `json:"logo_uri,omitempty"`
	Scope                             *string                         `json:"scope,omitempty"`
	Claims                            *string                         `json:"claims,omitempty"`
	Contacts                          *[]string                       `json:"contacts,omitempty"`
	TermsOfServiceURI                 *string                         `json:"tos_uri,omitempty"`
	PolicyURI                         *string                         `json:"policy_uri,omitempty"`
	JwksURI                           *string                         `json:"jwks_uri,omitempty"`
	Jwks                              json.RawMessage                 `json:"jwks,omitempty"`
	SoftwareID                        *string                         `json:"software_id,omitempty"`
	SoftwareVersion                   *string                         `json:"software_version,omitempty"`
	ForceAuthentication               *bool                           `json:"force_authentication,omitempty"`
	ForceConsent                      *bool                           `json:"force_consent,omitempty"`
	Audience                          *[]string                       `json:"audience,omitempty"`
	AttestationConfig                 *config.ClientAttestationConfig `json:"attestation_config,omitempty"`
	Public                            *bool                           `json:"public,omitempty"`
	ClearAttestationConfig            *bool                           `json:"clear_attestation_config,omitempty"`
}

type clientPersistResult struct {
	Response      ClientResponse
	PlainSecret   string
	RotatedSecret bool
}

// handlePutClient replaces all client metadata (RFC 7592-style).
// client_secret: if provided, rotates; if omitted, keeps the existing secret.
func (h *RegistrationHandler) handlePutClient(w http.ResponseWriter, r *http.Request, clientID string) {
	if clientID == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	if _, err := h.storage.GetClient(r.Context(), clientID); err != nil {
		h.log.Errorf("❌ [CLIENTS] PUT client not found: %s", clientID)
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	var metadata ClientMetadata
	if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
		h.log.Errorf("❌ [CLIENTS] PUT invalid body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	// Path is authoritative for client identity.
	metadata.ClientID = clientID

	rotateSecret := metadata.ClientSecret != ""
	result, err := h.persistClientMetadata(r.Context(), clientID, metadata, rotateSecret, true)
	if err != nil {
		h.writeClientPersistError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result.Response); err != nil {
		h.log.Errorf("❌ [CLIENTS] PUT encode failed: %v", err)
	}
	h.log.Printf("✅ [CLIENTS] PUT updated client %s (secret_rotated=%v)", clientID, result.RotatedSecret)
}

// handlePatchClient selectively updates client fields.
func (h *RegistrationHandler) handlePatchClient(w http.ResponseWriter, r *http.Request, clientID string) {
	if clientID == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	base, err := h.metadataFromExistingClient(r.Context(), clientID)
	if err != nil {
		h.log.Errorf("❌ [CLIENTS] PATCH client not found: %s", clientID)
		http.Error(w, "Client not found", http.StatusNotFound)
		return
	}

	var patch ClientPatchMetadata
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		h.log.Errorf("❌ [CLIENTS] PATCH invalid body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	metadata, rotateSecret, err := applyClientPatch(base, patch)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	metadata.ClientID = clientID

	result, err := h.persistClientMetadata(r.Context(), clientID, metadata, rotateSecret, false)
	if err != nil {
		h.writeClientPersistError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result.Response); err != nil {
		h.log.Errorf("❌ [CLIENTS] PATCH encode failed: %v", err)
	}
	h.log.Printf("✅ [CLIENTS] PATCH updated client %s (secret_rotated=%v)", clientID, result.RotatedSecret)
}

func applyClientPatch(base ClientMetadata, patch ClientPatchMetadata) (ClientMetadata, bool, error) {
	rotateSecret := false
	out := base

	if patch.ClientSecret != nil {
		if strings.TrimSpace(*patch.ClientSecret) == "" {
			return ClientMetadata{}, false, fmt.Errorf("client_secret cannot be empty when provided")
		}
		out.ClientSecret = *patch.ClientSecret
		rotateSecret = true
	}
	if patch.RedirectURIs != nil {
		out.RedirectURIs = *patch.RedirectURIs
	}
	if patch.TokenEndpointAuthMethod != nil {
		out.TokenEndpointAuthMethod = *patch.TokenEndpointAuthMethod
	}
	if patch.TokenEndpointAuthSigningAlgorithm != nil {
		out.TokenEndpointAuthSigningAlgorithm = *patch.TokenEndpointAuthSigningAlgorithm
	}
	if patch.GrantTypes != nil {
		out.GrantTypes = *patch.GrantTypes
	}
	if patch.ResponseTypes != nil {
		out.ResponseTypes = *patch.ResponseTypes
	}
	if patch.ClientName != nil {
		out.ClientName = *patch.ClientName
	}
	if patch.ClientURI != nil {
		out.ClientURI = *patch.ClientURI
	}
	if patch.LogoURI != nil {
		out.LogoURI = *patch.LogoURI
	}
	if patch.Scope != nil {
		out.Scope = *patch.Scope
	}
	if patch.Claims != nil {
		out.Claims = *patch.Claims
	}
	if patch.Contacts != nil {
		out.Contacts = *patch.Contacts
	}
	if patch.TermsOfServiceURI != nil {
		out.TermsOfServiceURI = *patch.TermsOfServiceURI
	}
	if patch.PolicyURI != nil {
		out.PolicyURI = *patch.PolicyURI
	}
	if patch.JwksURI != nil {
		out.JwksURI = *patch.JwksURI
	}
	if len(patch.Jwks) > 0 && string(patch.Jwks) != "null" {
		out.Jwks = patch.Jwks
	}
	if patch.SoftwareID != nil {
		out.SoftwareID = *patch.SoftwareID
	}
	if patch.SoftwareVersion != nil {
		out.SoftwareVersion = *patch.SoftwareVersion
	}
	if patch.ForceAuthentication != nil {
		out.ForceAuthentication = *patch.ForceAuthentication
	}
	if patch.ForceConsent != nil {
		out.ForceConsent = *patch.ForceConsent
	}
	if patch.Audience != nil {
		out.Audience = *patch.Audience
	}
	if patch.Public != nil {
		out.Public = *patch.Public
	}
	if patch.ClearAttestationConfig != nil && *patch.ClearAttestationConfig {
		out.AttestationConfig = nil
	} else if patch.AttestationConfig != nil {
		out.AttestationConfig = patch.AttestationConfig
	}

	return out, rotateSecret, nil
}

func (h *RegistrationHandler) metadataFromExistingClient(ctx context.Context, clientID string) (ClientMetadata, error) {
	client, err := h.storage.GetClient(ctx, clientID)
	if err != nil {
		return ClientMetadata{}, err
	}

	meta := ClientMetadata{
		ClientID:      clientID,
		RedirectURIs:  client.GetRedirectURIs(),
		GrantTypes:    append([]string(nil), client.GetGrantTypes()...),
		ResponseTypes: append([]string(nil), client.GetResponseTypes()...),
		Scope:         strings.Join(client.GetScopes(), " "),
		Audience:      append([]string(nil), client.GetAudience()...),
		Public:        client.IsPublic(),
	}

	if custom, ok := client.(*types.CustomClient); ok {
		meta.TokenEndpointAuthMethod = custom.GetTokenEndpointAuthMethod()
		meta.TokenEndpointAuthSigningAlgorithm = custom.TokenEndpointAuthSigningAlgorithm
		meta.JwksURI = custom.JSONWebKeysURI
		meta.ForceAuthentication = custom.ForceAuthentication
		meta.ForceConsent = custom.ForceConsent
		if len(custom.Claims) > 0 {
			meta.Claims = strings.Join(custom.Claims, " ")
		}
		if custom.JSONWebKeys != nil {
			if raw, err := json.Marshal(custom.JSONWebKeys); err == nil {
				meta.Jwks = raw
			}
		}
	} else if client.IsPublic() {
		meta.TokenEndpointAuthMethod = "none"
	} else {
		meta.TokenEndpointAuthMethod = "client_secret_basic"
	}

	if att, ok := GetClientAttestationConfig(ctx, clientID, h.storage); ok {
		meta.AttestationConfig = att
	}

	return meta, nil
}

func (h *RegistrationHandler) persistClientMetadata(ctx context.Context, clientID string, metadata ClientMetadata, rotateSecret bool, fullReplace bool) (*clientPersistResult, error) {
	isPublic := metadata.Public
	if !metadata.Public {
		isPublic = metadata.TokenEndpointAuthMethod == "none"
	}

	authMethod := metadata.TokenEndpointAuthMethod
	if authMethod == "" {
		if isPublic {
			authMethod = "none"
		} else {
			authMethod = "client_secret_basic"
		}
	}
	if authMethod == "none" {
		isPublic = true
	}

	grantTypes := metadata.GrantTypes
	if len(grantTypes) == 0 {
		if fullReplace {
			grantTypes = []string{"authorization_code"}
		} else {
			return nil, fmt.Errorf("grant_types cannot be empty")
		}
	}
	responseTypes := metadata.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}

	if authMethod == "private_key_jwt" && len(metadata.Jwks) == 0 && metadata.JwksURI == "" {
		return nil, fmt.Errorf("private_key_jwt requires jwks or jwks_uri")
	}
	if authMethod == "client_secret_jwt" && isPublic {
		return nil, fmt.Errorf("client_secret_jwt requires a confidential client with a secret")
	}

	parsedJWKS, err := auth.ParseClientJWKS(metadata.Jwks)
	if err != nil {
		return nil, fmt.Errorf("invalid jwks: %w", err)
	}

	var finalAttestationConfig *config.ClientAttestationConfig
	if authMethod == "attest_jwt_client_auth" || authMethod == "attest_tls_client_auth" {
		if metadata.AttestationConfig == nil {
			return nil, fmt.Errorf("attestation configuration required for attestation-based authentication")
		}
		if err := metadata.AttestationConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid attestation configuration: %w", err)
		}
		if metadata.AttestationConfig.ClientID != "" && metadata.AttestationConfig.ClientID != clientID {
			return nil, fmt.Errorf("attestation config client_id must match the registered client_id")
		}
		if metadata.AttestationConfig.ClientID == "" {
			metadata.AttestationConfig.ClientID = clientID
		}
		if err := h.resolveTrustAnchors(ctx, metadata.AttestationConfig); err != nil {
			return nil, err
		}
		finalAttestationConfig = metadata.AttestationConfig

		hasClientCredentials := false
		for _, gt := range grantTypes {
			if gt == "client_credentials" {
				hasClientCredentials = true
				break
			}
		}
		if !hasClientCredentials {
			grantTypes = append(grantTypes, "client_credentials")
		}
	} else if metadata.AttestationConfig != nil {
		// Allow storing attestation config even if auth method is not attestation-based.
		finalAttestationConfig = metadata.AttestationConfig
		if finalAttestationConfig.ClientID == "" {
			finalAttestationConfig.ClientID = clientID
		}
	}

	var scopes []string
	if metadata.Scope != "" {
		scopes = splitScope(metadata.Scope)
	}
	if !contains(scopes, "openid") {
		scopes = append(scopes, "openid")
	}

	// client_id is not auto-added; include it in metadata.Audience when needed.
	audience := append([]string(nil), metadata.Audience...)
	if h.config != nil && h.config.Security.PrivilegedClientID != "" && !contains(audience, h.config.Security.PrivilegedClientID) {
		audience = append(audience, h.config.Security.PrivilegedClientID)
	}

	var claims []string
	if metadata.Claims != "" {
		claims = splitScope(metadata.Claims)
	}

	existing, err := h.storage.GetClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	var (
		clientSecret string
		hashedSecret []byte
		rotated      bool
	)

	switch {
	case isPublic:
		hashedSecret = nil
		clientSecret = ""
	case rotateSecret:
		clientSecret = metadata.ClientSecret
		if clientSecret == "" {
			return nil, fmt.Errorf("client_secret required when rotating secret")
		}
		hashedSecret, err = utils.HashSecret(clientSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to hash client secret: %w", err)
		}
		rotated = true
	default:
		hashedSecret = existing.GetHashedSecret()
		if secret, ok := GetClientSecret(ctx, clientID, h.storage, h.secretManager); ok {
			clientSecret = secret
		}
	}

	signingAlg := metadata.TokenEndpointAuthSigningAlgorithm
	if signingAlg == "" {
		switch authMethod {
		case "client_secret_jwt":
			signingAlg = "HS256"
		case "private_key_jwt":
			signingAlg = "RS256"
		}
	}

	var secretBytes []byte
	if !isPublic {
		secretBytes = hashedSecret
	}

	newClient := &types.CustomClient{
		DefaultClient: &fosite.DefaultClient{
			ID:            clientID,
			Secret:        secretBytes,
			RedirectURIs:  metadata.RedirectURIs,
			GrantTypes:    grantTypes,
			ResponseTypes: responseTypes,
			Scopes:        scopes,
			Audience:      audience,
			Public:        isPublic,
		},
		Claims:                            claims,
		ForceAuthentication:               metadata.ForceAuthentication,
		ForceConsent:                      metadata.ForceConsent,
		JSONWebKeysURI:                    metadata.JwksURI,
		JSONWebKeys:                       parsedJWKS,
		TokenEndpointAuthMethod:           authMethod,
		TokenEndpointAuthSigningAlgorithm: signingAlg,
	}

	if err := h.storage.UpdateClient(ctx, clientID, newClient); err != nil {
		return nil, fmt.Errorf("failed to update client: %w", err)
	}

	if isPublic {
		_ = h.storage.DeleteClientSecret(ctx, clientID)
	} else if rotated {
		encryptedSecret, err := h.secretManager.EncryptSecret(clientSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
		}
		if err := h.storage.StoreClientSecret(ctx, clientID, encryptedSecret); err != nil {
			return nil, fmt.Errorf("failed to store client secret: %w", err)
		}
	}

	if finalAttestationConfig != nil {
		if h.attestationManager != nil {
			h.attestationManager.AddClientConfig(clientID, finalAttestationConfig)
		}
		if err := h.storage.StoreAttestationConfig(ctx, clientID, finalAttestationConfig); err != nil {
			return nil, fmt.Errorf("failed to store attestation config: %w", err)
		}
	} else {
		_ = h.storage.DeleteAttestationConfig(ctx, clientID)
	}

	responseSecret := ""
	if !isPublic {
		// Only echo plaintext secret when rotated; otherwise omit (caller already knows it).
		if rotated {
			responseSecret = clientSecret
		}
	}

	resp := ClientResponse{
		ClientID:                          clientID,
		ClientSecret:                      responseSecret,
		ClientSecretExpiresAt:             0,
		ClientIdIssuedAt:                  0,
		RedirectURIs:                      metadata.RedirectURIs,
		TokenEndpointAuthMethod:           authMethod,
		TokenEndpointAuthSigningAlgorithm: signingAlg,
		GrantTypes:                        grantTypes,
		ResponseTypes:                     responseTypes,
		ClientName:                        metadata.ClientName,
		ClientURI:                         metadata.ClientURI,
		LogoURI:                           metadata.LogoURI,
		Scope:                             strings.Join(scopes, " "),
		Claims:                            strings.Join(claims, " "),
		Contacts:                          metadata.Contacts,
		TermsOfServiceURI:                 metadata.TermsOfServiceURI,
		PolicyURI:                         metadata.PolicyURI,
		JwksURI:                           metadata.JwksURI,
		Jwks:                              metadata.Jwks,
		SoftwareID:                        metadata.SoftwareID,
		SoftwareVersion:                   metadata.SoftwareVersion,
		ForceAuthentication:               metadata.ForceAuthentication,
		ForceConsent:                      metadata.ForceConsent,
		Audience:                          audience,
		AttestationConfig:                 finalAttestationConfig,
		Public:                            isPublic,
	}

	return &clientPersistResult{
		Response:      resp,
		PlainSecret:   clientSecret,
		RotatedSecret: rotated,
	}, nil
}

type clientPersistHTTPError struct {
	Status  int
	Message string
}

func (e *clientPersistHTTPError) Error() string { return e.Message }

func (h *RegistrationHandler) writeClientPersistError(w http.ResponseWriter, err error) {
	if httpErr, ok := err.(*clientPersistHTTPError); ok {
		http.Error(w, httpErr.Message, httpErr.Status)
		return
	}
	msg := err.Error()
	status := http.StatusBadRequest
	if strings.HasPrefix(msg, "failed to ") {
		status = http.StatusInternalServerError
	}
	h.log.Errorf("❌ [CLIENTS] persist error: %v", err)
	http.Error(w, msg, status)
}
