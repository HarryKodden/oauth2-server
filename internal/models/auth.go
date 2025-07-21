package models

// ClientCredentialsRequest represents a client credentials request
type ClientCredentialsRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scope        string `json:"scope,omitempty"`
}

// ClientCredentialsResponse represents a client credentials response
type ClientCredentialsResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// ClientAuthRequest represents a client authentication request
type ClientAuthRequest struct {
    ClientID     string `json:"client_id"`
    ClientSecret string `json:"client_secret"`
}

// ClientAuthResponse represents a client authentication response
type ClientAuthResponse struct {
    ClientID      string   `json:"client_id"`
    Scopes        []string `json:"scopes,omitempty"`
    GrantTypes    []string `json:"grant_types,omitempty"`
    Audience      []string `json:"audience,omitempty"`
    Authenticated bool     `json:"authenticated"`
}
