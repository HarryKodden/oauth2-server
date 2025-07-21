package flows

import (
	"encoding/json"
	"log"
	"net/http"

	"oauth2-server/internal/auth"
	"oauth2-server/internal/models"
	"oauth2-server/internal/store"
	"oauth2-server/internal/utils"
	"oauth2-server/pkg/config"
)

// TokenExchangeFlow handles RFC 8693 token exchange
type TokenExchangeFlow struct {
	clientStore *store.ClientStore
	tokenStore  *store.TokenStore
	config      *config.Config
}

// NewTokenExchangeFlow creates a new token exchange flow handler
func NewTokenExchangeFlow(clientStore *store.ClientStore, tokenStore *store.TokenStore, cfg *config.Config) *TokenExchangeFlow {
	return &TokenExchangeFlow{
		clientStore: clientStore,
		tokenStore:  tokenStore,
		config:      cfg,
	}
}

// Handle processes token exchange requests
func (f *TokenExchangeFlow) Handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Processing token exchange request")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		f.writeError(w, "invalid_request", "Failed to parse request")
		return
	}

	grantType := r.FormValue("grant_type")
	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	requestedTokenType := r.FormValue("requested_token_type")
	scope := r.FormValue("scope")

	if grantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		f.writeError(w, "unsupported_grant_type", "Grant type must be token-exchange")
		return
	}

	if subjectToken == "" {
		f.writeError(w, "invalid_request", "subject_token is required")
		return
	}

	if subjectTokenType == "" {
		f.writeError(w, "invalid_request", "subject_token_type is required")
		return
	}

	// Extract client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		f.writeError(w, "invalid_client", "Client authentication required")
		return
	}

	// Authenticate client
	client, err := auth.AuthenticateClient(clientID, clientSecret, f.clientStore)
	if err != nil {
		log.Printf("❌ Client authentication failed for %s: %v", clientID, err)
		f.writeError(w, "invalid_client", "Client authentication failed")
		return
	}

	// Check if client is authorized for token exchange
	if !auth.ClientHasGrantType(client, "urn:ietf:params:oauth:grant-type:token-exchange") {
		f.writeError(w, "unauthorized_client", "Client not authorized for token exchange")
		return
	}

	// Validate subject token
	if err := auth.ValidateAccessToken(subjectToken); err != nil {
		f.writeError(w, "invalid_grant", "Invalid subject token")
		return
	}

	// Validate token type
	if subjectTokenType != "urn:ietf:params:oauth:token-type:access_token" {
		f.writeError(w, "invalid_request", "Unsupported subject token type")
		return
	}

	// Generate new access token (no parameters)
	newAccessToken := utils.GenerateAccessToken()

	// Create response
	response := models.TokenResponse{
		AccessToken: newAccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
		Scope:       scope,
	}

	// Set issued token type if requested
	if requestedTokenType != "" {
		// Add issued_token_type to response if needed
		// This might require extending the TokenResponse model
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Token exchanged for client: %s", clientID)
}

func (f *TokenExchangeFlow) writeError(w http.ResponseWriter, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	errorResp := models.ErrorResponse{
		Error:            errorCode,
		ErrorDescription: description,
	}
	json.NewEncoder(w).Encode(errorResp)
}