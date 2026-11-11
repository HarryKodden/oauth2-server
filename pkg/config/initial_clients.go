package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// initialClientJSON is the JSON shape accepted by INITIAL_CLIENTS.
// Field names mirror ClientConfig / dynamic registration where practical.
type initialClientJSON struct {
	ID                                string                   `json:"id"`
	ClientID                          string                   `json:"client_id"`
	Secret                            string                   `json:"secret"`
	ClientSecret                      string                   `json:"client_secret"`
	Name                              string                   `json:"name"`
	Description                       string                   `json:"description"`
	RedirectURIs                      []string                 `json:"redirect_uris"`
	GrantTypes                        []string                 `json:"grant_types"`
	ResponseTypes                     []string                 `json:"response_types"`
	Scopes                            []string                 `json:"scopes"`
	Claims                            []string                 `json:"claims"`
	ForceAuthentication               bool                     `json:"force_authentication"`
	ForceConsent                      bool                     `json:"force_consent"`
	Audience                          []string                 `json:"audience"`
	TokenEndpointAuthMethod           string                   `json:"token_endpoint_auth_method"`
	TokenEndpointAuthSigningAlgorithm string                   `json:"token_endpoint_auth_signing_alg"`
	JwksURI                           string                   `json:"jwks_uri"`
	Jwks                              json.RawMessage          `json:"jwks"`
	Public                            bool                     `json:"public"`
	EnabledFlows                      []string                 `json:"enabled_flows"`
	Enabled                           *bool                    `json:"enabled"`
	AttestationConfig                 *ClientAttestationConfig `json:"attestation_config"`
}

// ParseInitialClientsJSON parses INITIAL_CLIENTS multiline JSON.
// Accepted forms:
//  1. JSON array:  [ { "id": "webapp", ... }, ... ]
//  2. JSON object: { "webapp": { "secret": "...", ... }, ... }
//     Object keys are used as client IDs when the body omits id/client_id.
func ParseInitialClientsJSON(raw string) ([]ClientConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	switch raw[0] {
	case '[':
		var items []initialClientJSON
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return nil, fmt.Errorf("INITIAL_CLIENTS: invalid JSON array: %w", err)
		}
		out := make([]ClientConfig, 0, len(items))
		for i, item := range items {
			client, err := item.toClientConfig("")
			if err != nil {
				return nil, fmt.Errorf("INITIAL_CLIENTS[%d]: %w", i, err)
			}
			out = append(out, client)
		}
		return out, nil

	case '{':
		// Try object-map form first: { "client-id": { ... }, ... }
		var asMap map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &asMap); err != nil {
			return nil, fmt.Errorf("INITIAL_CLIENTS: invalid JSON object: %w", err)
		}
		if len(asMap) == 0 {
			return nil, nil
		}

		// Detect a single client object (has typical client fields) vs an id->client map.
		if looksLikeSingleClientObject(asMap) {
			var item initialClientJSON
			if err := json.Unmarshal([]byte(raw), &item); err != nil {
				return nil, fmt.Errorf("INITIAL_CLIENTS: invalid client object: %w", err)
			}
			client, err := item.toClientConfig("")
			if err != nil {
				return nil, fmt.Errorf("INITIAL_CLIENTS: %w", err)
			}
			return []ClientConfig{client}, nil
		}

		out := make([]ClientConfig, 0, len(asMap))
		for id, body := range asMap {
			var item initialClientJSON
			if err := json.Unmarshal(body, &item); err != nil {
				return nil, fmt.Errorf("INITIAL_CLIENTS[%s]: invalid client object: %w", id, err)
			}
			client, err := item.toClientConfig(id)
			if err != nil {
				return nil, fmt.Errorf("INITIAL_CLIENTS[%s]: %w", id, err)
			}
			out = append(out, client)
		}
		return out, nil

	default:
		return nil, fmt.Errorf("INITIAL_CLIENTS: expected JSON array or object, got %q", raw[:1])
	}
}

func looksLikeSingleClientObject(fields map[string]json.RawMessage) bool {
	clientFieldHints := []string{
		"id", "client_id", "secret", "client_secret", "redirect_uris",
		"grant_types", "token_endpoint_auth_method", "jwks", "jwks_uri", "public",
	}
	for _, key := range clientFieldHints {
		if _, ok := fields[key]; ok {
			return true
		}
	}
	return false
}

func (item initialClientJSON) toClientConfig(fallbackID string) (ClientConfig, error) {
	id := item.ID
	if id == "" {
		id = item.ClientID
	}
	if id == "" {
		id = fallbackID
	}
	if id == "" {
		return ClientConfig{}, fmt.Errorf("client id is required")
	}

	secret := item.Secret
	if secret == "" {
		secret = item.ClientSecret
	}

	authMethod := item.TokenEndpointAuthMethod
	if authMethod == "" {
		if item.Public {
			authMethod = "none"
		} else if item.JwksURI != "" || len(item.Jwks) > 0 {
			authMethod = "private_key_jwt"
		} else {
			authMethod = "client_secret_basic"
		}
	}

	grantTypes := item.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	responseTypes := item.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	scopes := item.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email"}
	}
	enabledFlows := item.EnabledFlows
	if len(enabledFlows) == 0 {
		enabledFlows = grantTypes
	}

	jwks := ""
	if len(item.Jwks) > 0 && string(item.Jwks) != "null" {
		jwks = string(item.Jwks)
	}

	name := item.Name
	if name == "" {
		name = "Initial Client " + id
	}

	if item.AttestationConfig != nil && item.AttestationConfig.ClientID == "" {
		item.AttestationConfig.ClientID = id
	}

	return ClientConfig{
		ID:                                id,
		Secret:                            secret,
		Name:                              name,
		Description:                       item.Description,
		RedirectURIs:                      item.RedirectURIs,
		GrantTypes:                        grantTypes,
		ResponseTypes:                     responseTypes,
		Scopes:                            scopes,
		Claims:                            item.Claims,
		ForceAuthentication:               item.ForceAuthentication,
		ForceConsent:                      item.ForceConsent,
		Audience:                          item.Audience,
		TokenEndpointAuthMethod:           authMethod,
		TokenEndpointAuthSigningAlgorithm: item.TokenEndpointAuthSigningAlgorithm,
		JwksURI:                           item.JwksURI,
		Jwks:                              jwks,
		Public:                            item.Public || authMethod == "none",
		EnabledFlows:                      enabledFlows,
		Enabled:                           item.Enabled,
		AttestationConfig:                 item.AttestationConfig,
	}, nil
}

// mergeClientsByID upserts clients into the existing slice (matched by ID).
func mergeClientsByID(existing []ClientConfig, incoming []ClientConfig) []ClientConfig {
	if len(incoming) == 0 {
		return existing
	}
	index := make(map[string]int, len(existing))
	for i, client := range existing {
		index[client.ID] = i
	}
	for _, client := range incoming {
		if i, ok := index[client.ID]; ok {
			existing[i] = client
			continue
		}
		index[client.ID] = len(existing)
		existing = append(existing, client)
	}
	return existing
}
