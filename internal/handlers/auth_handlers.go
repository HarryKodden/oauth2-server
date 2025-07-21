package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"oauth2-server/internal/auth"
	"oauth2-server/internal/models"
	"oauth2-server/internal/store"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	clientStore *store.ClientStore
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(clientStore *store.ClientStore) *AuthHandler {
	return &AuthHandler{
		clientStore: clientStore,
	}
}

// HandleClientAuth handles client authentication requests
func (h *AuthHandler) HandleClientAuth(w http.ResponseWriter, r *http.Request) {
	var req models.ClientAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("❌ Failed to decode client auth request: %v", err)
		h.writeError(w, "invalid_request", "Invalid request format")
		return
	}

	// Authenticate client using the store
	client, err := auth.AuthenticateClient(req.ClientID, req.ClientSecret, h.clientStore)
	if err != nil {
		log.Printf("❌ Client authentication failed for %s: %v", req.ClientID, err)
		h.writeError(w, "invalid_client", "Client authentication failed")
		return
	}

	response := models.ClientAuthResponse{
		ClientID:     client.GetID(),
		Scopes:       client.GetScopes(),
		GrantTypes:   client.GetGrantTypes(),
		Audience:     client.GetAudience(),
		Authenticated: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Client %s authenticated successfully", req.ClientID)
}

// HandleTokenValidation handles token validation requests
func (h *AuthHandler) HandleTokenValidation(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		h.writeError(w, "invalid_request", "Authorization header required")
		return
	}

	// Extract Bearer token
	token, err := auth.ExtractBearerToken(authHeader)
	if err != nil {
		h.writeError(w, "invalid_token", "Invalid authorization header format")
		return
	}

	// Validate token
	if err := auth.ValidateAccessToken(token); err != nil {
		log.Printf("❌ Token validation failed: %v", err)
		h.writeError(w, "invalid_token", "Token validation failed")
		return
	}

	// Token is valid
	response := models.TokenValidationResponse{
		Valid:  true,
		Active: true,
		Token:  token[:20] + "...", // Truncated for security
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Token validated successfully")
}

// HandleIntrospection handles token introspection requests (RFC 7662)
func (h *AuthHandler) HandleIntrospection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeError(w, "invalid_request", "Failed to parse request")
		return
	}

	token := r.FormValue("token")
	if token == "" {
		h.writeError(w, "invalid_request", "Token parameter is required")
		return
	}

	// Authenticate the client making the introspection request
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		h.writeError(w, "invalid_client", "Client authentication required")
		return
	}

	_, err = auth.AuthenticateClient(clientID, clientSecret, h.clientStore)
	if err != nil {
		h.writeError(w, "invalid_client", "Client authentication failed")
		return
	}

	// Validate the token being introspected
	introspectionResp := models.IntrospectionResponse{
		Active: false,
	}

	if err := auth.ValidateAccessToken(token); err == nil {
		// Token is valid and active
		introspectionResp = models.IntrospectionResponse{
			Active:    true,
			TokenType: "Bearer",
			Scope:     "api:read api:write",
			ClientID:  clientID,
			Username:  "user123", // This would come from token claims in real implementation
			Exp:       1234567890, // This would come from token claims
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(introspectionResp)
}

// HandleRevocation handles token revocation requests (RFC 7009)
func (h *AuthHandler) HandleRevocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.writeError(w, "invalid_request", "Failed to parse request")
		return
	}

	token := r.FormValue("token")
	if token == "" {
		h.writeError(w, "invalid_request", "Token parameter is required")
		return
	}

	// Authenticate the client making the revocation request
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		h.writeError(w, "invalid_client", "Client authentication required")
		return
	}

	_, err = auth.AuthenticateClient(clientID, clientSecret, h.clientStore)
	if err != nil {
		h.writeError(w, "invalid_client", "Client authentication failed")
		return
	}

	// In a real implementation, you would revoke the token from your token store
	// For now, we'll just return success
	log.Printf("🗑️ Token revoked for client: %s", clientID)

	// RFC 7009 specifies that revocation should return 200 OK even if token was invalid
	w.WriteHeader(http.StatusOK)
}

// writeError writes an OAuth2 error response
func (h *AuthHandler) writeError(w http.ResponseWriter, errorCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	errorResp := models.ErrorResponse{
		Error:            errorCode,
		ErrorDescription: description,
	}

	json.NewEncoder(w).Encode(errorResp)
}