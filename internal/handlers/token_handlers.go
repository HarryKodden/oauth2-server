package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"oauth2-server/internal/auth"
	"oauth2-server/internal/store"
	"oauth2-server/internal/utils"
	"oauth2-server/pkg/config"

	"github.com/ory/fosite"
)

// TokenHandlers handles token-related endpoints
type TokenHandlers struct {
	clientStore *store.ClientStore
	tokenStore  *store.TokenStore
	config      *config.Config
}

// NewTokenHandlers creates a new token handlers instance
func NewTokenHandlers(clientStore *store.ClientStore, tokenStore *store.TokenStore, cfg *config.Config) *TokenHandlers {
	return &TokenHandlers{
		clientStore: clientStore,
		tokenStore:  tokenStore,
		config:      cfg,
	}
}

// HandleTokenRevocation handles token revocation requests (RFC 7009)
func (h *TokenHandlers) HandleTokenRevocation(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Processing token revocation request")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	if err := r.ParseForm(); err != nil {
		utils.WriteErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}

	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")

	if token == "" {
		utils.WriteErrorResponse(w, "invalid_request", "token is required")
		return
	}

	// Extract client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication required")
		return
	}

	// Authenticate client
	_, err = auth.AuthenticateClient(clientID, clientSecret, h.clientStore)
	if err != nil {
		log.Printf("❌ Client authentication failed for %s: %v", clientID, err)
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}

	// Validate that the token belongs to the client
	tokenInfo, err := h.tokenStore.ValidateAccessToken(token)
	if err != nil {
		// Try refresh token if access token validation fails
		tokenInfo, err = h.tokenStore.ValidateRefreshToken(token)
	}

	// Fix: Check for error instead of using ! operator on interface
	if err != nil {
		// Token not found or invalid - per RFC 7009, we should return success anyway
		log.Printf("⚠️ Token not found or invalid: %v", err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify token belongs to the client
	if tokenInfo.ClientID != clientID {
		log.Printf("⚠️ Token does not belong to client %s", clientID)
		w.WriteHeader(http.StatusOK) // Per RFC 7009, return success even if token doesn't belong to client
		return
	}

	// Revoke the token
	err = h.tokenStore.RevokeToken(token)
	if err != nil {
		log.Printf("❌ Failed to revoke token: %v", err)
		utils.WriteServerError(w, "Failed to revoke token")
		return
	}

	// Handle hint for token type (optional optimization)
	if tokenTypeHint == "refresh_token" {
		// Also revoke associated access tokens if this is a refresh token
		// In a production system, you'd track these relationships
	}

	log.Printf("✅ Token revoked for client: %s", clientID)
	w.WriteHeader(http.StatusOK)
}

// HandleTokenIntrospection handles token introspection requests (RFC 7662)
func (h *TokenHandlers) HandleTokenIntrospection(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔍 Processing token introspection request")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	if err := r.ParseForm(); err != nil {
		utils.WriteErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}

	token := r.FormValue("token")
	tokenTypeHint := r.FormValue("token_type_hint")

	if token == "" {
		utils.WriteErrorResponse(w, "invalid_request", "token is required")
		return
	}

	// Extract client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication required")
		return
	}

	// Authenticate client
	_, err = auth.AuthenticateClient(clientID, clientSecret, h.clientStore)
	if err != nil {
		log.Printf("❌ Client authentication failed for %s: %v", clientID, err)
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}

	// Validate token and get info
	var tokenInfo *store.TokenInfo
	if tokenTypeHint == "refresh_token" {
		tokenInfo, err = h.tokenStore.ValidateRefreshToken(token)
	} else {
		// Default to access token or try both
		tokenInfo, err = h.tokenStore.ValidateAccessToken(token)
		if err != nil {
			tokenInfo, err = h.tokenStore.ValidateRefreshToken(token)
		}
	}

	if err != nil {
		// Token is invalid or expired
		response := map[string]interface{}{
			"active": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create introspection response
	response := map[string]interface{}{
		"active":     tokenInfo.Active,
		"token_type": tokenInfo.TokenType,
		"client_id":  tokenInfo.ClientID,
		"username":   tokenInfo.UserID,
		"exp":        tokenInfo.ExpiresAt.Unix(),
		"iat":        tokenInfo.IssuedAt.Unix(),
		"iss":        tokenInfo.Issuer,
		"aud":        tokenInfo.Audience,
	}

	// Fix: Use tokenInfo.Scopes (slice) and join them
	if len(tokenInfo.Scopes) > 0 {
		response["scope"] = strings.Join(tokenInfo.Scopes, " ")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Token introspection completed for client: %s", clientID)
}

// handleTokenExchange processes token exchange requests (RFC 8693)
func (h *TokenHandlers) HandleTokenExchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		utils.WriteMethodNotAllowedError(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.WriteInvalidRequestError(w, "Failed to parse request")
		return
	}

	// Extract and validate client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		utils.WriteInvalidClientError(w, "Client authentication required")
		return
	}

	// Authenticate client
	if err := h.clientStore.ValidateClientCredentials(clientID, clientSecret); err != nil {
		utils.WriteInvalidClientError(w, "Invalid client credentials")
		return
	}

	// Validate required parameters
	subjectToken := r.FormValue("subject_token")
	subjectTokenType := r.FormValue("subject_token_type")
	audience := r.FormValue("audience")

	if subjectToken == "" {
		utils.WriteInvalidRequestError(w, "subject_token is required")
		return
	}

	if subjectTokenType == "" {
		utils.WriteInvalidRequestError(w, "subject_token_type is required")
		return
	}

	// Validate subject token type
	if subjectTokenType != "urn:ietf:params:oauth:token-type:access_token" &&
		subjectTokenType != "urn:ietf:params:oauth:token-type:refresh_token" &&
		subjectTokenType != "urn:ietf:params:oauth:token-type:id_token" {
		utils.WriteUnsupportedGrantTypeError(w, "Unsupported subject_token_type")
		return
	}

	// Validate the subject token
	tokenInfo, err := h.tokenStore.ValidateAccessToken(subjectToken)
	if err != nil {
		utils.WriteInvalidGrantError(w, "Invalid or expired subject_token")
		return
	}

	// Optional: Validate audience if provided
	if audience != "" && !h.validateAudience(clientID, audience) {
		utils.WriteInvalidRequestError(w, "Invalid audience")
		return
	}

	// Generate new access token
	newAccessToken, err := h.generateAccessToken()
	if err != nil {
		utils.WriteServerError(w, "Failed to generate access token")
		return
	}

	// Determine the scope for the new token
	requestedScope := r.FormValue("scope")
	// Fix: Use tokenInfo.Scopes (slice) and join them
	originalScope := strings.Join(tokenInfo.Scopes, " ")
	scope := h.determineTokenExchangeScope(originalScope, requestedScope)

	// Store the new token - Fix: Pass []string for scopes and time.Time for expiry
	scopeSlice := strings.Fields(scope)
	expiresAt := time.Now().Add(time.Hour)
	h.tokenStore.StoreAccessToken(newAccessToken, clientID, tokenInfo.UserID, scopeSlice, expiresAt)

	// Prepare response
	response := map[string]interface{}{
		"access_token":      newAccessToken,
		"token_type":        "Bearer",
		"expires_in":        3600,
		"scope":             scope,
		"issued_token_type": "urn:ietf:params:oauth:token-type:access_token",
	}

	// Optional: Include refresh token if appropriate
	if strings.Contains(scope, "offline_access") {
		refreshToken, err := h.generateRefreshToken()
		if err == nil {
			refreshExpiresAt := time.Now().Add(24 * time.Hour * 30)
			scopeSlice := strings.Fields(scope)
			h.tokenStore.StoreRefreshToken(refreshToken, clientID, tokenInfo.UserID, scopeSlice, refreshExpiresAt)
			response["refresh_token"] = refreshToken
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Token exchange completed for client: %s, user: %s", clientID, tokenInfo.UserID)
}

// handleClientCredentials processes client credentials requests (RFC 6749 Section 4.4)
func (h *TokenHandlers) HandleClientCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		utils.WriteMethodNotAllowedError(w)
		return
	}

	if err := r.ParseForm(); err != nil {
		utils.WriteInvalidRequestError(w, "Failed to parse request")
		return
	}

	// Extract and validate client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		utils.WriteInvalidClientError(w, "Client authentication required")
		return
	}

	// Authenticate client
	client, err := h.clientStore.GetClient(r.Context(), clientID)
	if err != nil {
		utils.WriteInvalidClientError(w, "Invalid client")
		return
	}

	if err := h.clientStore.ValidateClientCredentials(clientID, clientSecret); err != nil {
		utils.WriteInvalidClientError(w, "Invalid client credentials")
		return
	}

	// Check if client is authorized for client_credentials grant
	if !h.clientSupportsGrantType(client, "client_credentials") {
		utils.WriteInvalidRequestError(w, "Client not authorized for client_credentials grant")
		return
	}

	// Validate requested scope
	requestedScope := r.FormValue("scope")
	if requestedScope == "" {
		requestedScope = strings.Join(client.GetScopes(), " ")
	}

	// Validate that the requested scope is allowed for this client
	if !h.validateClientScope(client, requestedScope) {
		utils.WriteInvalidScopeError(w, "Requested scope exceeds client permissions")
		return
	}

	// Generate access token
	accessToken, err := h.generateAccessToken()
	if err != nil {
		utils.WriteServerError(w, "Failed to generate access token")
		return
	}

	// Store the token (no user ID for client credentials)
	// Fix: Pass []string for scopes and time.Time for expiry
	scopeSlice := strings.Fields(requestedScope)
	expiresAt := time.Now().Add(time.Hour)
	h.tokenStore.StoreAccessToken(accessToken, clientID, "", scopeSlice, expiresAt)

	// Prepare response
	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"scope":        requestedScope,
	}

	// Generate refresh token if offline_access scope is requested
	// This is useful for long-running services that need refresh capabilities
	if strings.Contains(requestedScope, "offline_access") || strings.Contains(requestedScope, "refresh_token") {
		refreshToken, err := h.generateRefreshToken()
		if err != nil {
			log.Printf("Failed to generate refresh token: %v", err)
		} else {
			// Store refresh token with expiry (30 days for refresh tokens)
			refreshExpiresAt := time.Now().Add(30 * 24 * time.Hour)
			h.tokenStore.StoreRefreshToken(refreshToken, clientID, "", scopeSlice, refreshExpiresAt)
			response["refresh_token"] = refreshToken
			log.Printf("✅ Refresh token issued for client credentials flow: %s", clientID)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(response)

	log.Printf("✅ Client credentials token issued for client: %s", clientID)
}

// HandleRefreshToken handles token refresh requests
func (h *TokenHandlers) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Processing token refresh request")

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	if err := r.ParseForm(); err != nil {
		utils.WriteErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}

	refreshToken := r.FormValue("refresh_token")
	scope := r.FormValue("scope")

	if refreshToken == "" {
		utils.WriteErrorResponse(w, "invalid_request", "refresh_token is required")
		return
	}

	// Extract client credentials
	clientID, clientSecret, err := auth.ExtractClientCredentials(r)
	if err != nil {
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication required")
		return
	}

	// Authenticate client
	_, err = auth.AuthenticateClient(clientID, clientSecret, h.clientStore)
	if err != nil {
		log.Printf("❌ Client authentication failed for %s: %v", clientID, err)
		utils.WriteErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}

	// Validate refresh token
	tokenInfo, err := h.tokenStore.ValidateRefreshToken(refreshToken)
	// Fix: Check for error instead of using ! operator on interface
	if err != nil {
		utils.WriteErrorResponse(w, "invalid_grant", "Invalid or expired refresh token")
		return
	}

	// Verify token belongs to the client
	if tokenInfo.ClientID != clientID {
		utils.WriteErrorResponse(w, "invalid_grant", "Refresh token does not belong to client")
		return
	}

	// Handle scope parameter
	var requestedScope string
	if scope != "" {
		// If scope is provided, it must be a subset of the original scope
		originalScope := strings.Join(tokenInfo.Scopes, " ")
		if !h.isScopeSubset(scope, originalScope) {
			utils.WriteErrorResponse(w, "invalid_scope", "Requested scope exceeds original scope")
			return
		}
		requestedScope = scope
	} else {
		// Use tokenInfo.Scopes (slice) and join them
		requestedScope = strings.Join(tokenInfo.Scopes, " ")
	}

	// Generate new access token
	requestedScopes := strings.Fields(requestedScope)
	newAccessToken, err := auth.GenerateAccessToken(tokenInfo.UserID, clientID, requestedScopes)
	if err != nil {
		log.Printf("❌ Error generating access token: %v", err)
		utils.WriteServerError(w, "Failed to generate access token")
		return
	}

	// Generate new refresh token
	newRefreshToken, err := auth.GenerateRefreshToken(tokenInfo.UserID, clientID)
	if err != nil {
		log.Printf("❌ Error generating refresh token: %v", err)
		utils.WriteServerError(w, "Failed to generate refresh token")
		return
	}

	// Store new tokens with correct parameters
	accessTokenExpiry := time.Now().Add(time.Hour)
	refreshTokenExpiry := time.Now().Add(24 * time.Hour)

	// Pass []string for scopes and time.Time for expiry
	err = h.tokenStore.StoreAccessToken(newAccessToken, clientID, tokenInfo.UserID, requestedScopes, accessTokenExpiry)
	if err != nil {
		log.Printf("❌ Error storing access token: %v", err)
		utils.WriteServerError(w, "Failed to store access token")
		return
	}

	// Store new refresh token with scopes
	err = h.tokenStore.StoreRefreshToken(newRefreshToken, clientID, tokenInfo.UserID, requestedScopes, refreshTokenExpiry)
	if err != nil {
		log.Printf("❌ Error storing refresh token: %v", err)
		utils.WriteServerError(w, "Failed to store refresh token")
		return
	}

	// Revoke old refresh token
	err = h.tokenStore.RevokeToken(refreshToken)
	if err != nil {
		log.Printf("⚠️ Warning: Failed to revoke old refresh token: %v", err)
		// Continue anyway as new tokens are already issued
	}

	// Create response
	response := map[string]interface{}{
		"access_token":  newAccessToken,
		"token_type":    "Bearer",
		"expires_in":    3600, // 1 hour
		"refresh_token": newRefreshToken,
		"scope":         requestedScope,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Tokens refreshed for client: %s", clientID)
}

// Helper functions

// generateAccessToken generates a random access token
func (h *TokenHandlers) generateAccessToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("access_%x", bytes), nil
}

// generateRefreshToken generates a random refresh token
func (h *TokenHandlers) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("refresh_%x", bytes), nil
}

// clientSupportsGrantType checks if a client supports a specific grant type
func (h *TokenHandlers) clientSupportsGrantType(client interface{}, grantType string) bool {
	// Try fosite.Arguments first (our client store)
	if c, ok := client.(interface{ GetGrantTypes() fosite.Arguments }); ok {
		for _, gt := range c.GetGrantTypes() {
			if gt == grantType {
				return true
			}
		}
		return false
	}
	// Fallback to []string interface
	if c, ok := client.(interface{ GetGrantTypes() []string }); ok {
		for _, gt := range c.GetGrantTypes() {
			if gt == grantType {
				return true
			}
		}
	}
	return false
}

// validateClientScope validates that requested scope is allowed for the client
func (h *TokenHandlers) validateClientScope(client interface{}, requestedScope string) bool {
	if requestedScope == "" {
		return true
	}

	if c, ok := client.(interface{ GetScopes() []string }); ok {
		clientScopes := c.GetScopes()
		requestedScopes := strings.Split(requestedScope, " ")

		for _, reqScope := range requestedScopes {
			if reqScope == "" {
				continue
			}
			found := false
			for _, clientScope := range clientScopes {
				if clientScope == reqScope {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

// validateAudience validates the audience parameter for token exchange
func (h *TokenHandlers) validateAudience(clientID, audience string) bool {
	// Get client and check if the audience is in the client's allowed audiences
	client, err := h.clientStore.GetClient(nil, clientID)
	if err != nil {
		return false
	}

	// Use type switch to handle different client types
	switch c := client.(type) {
	case interface{ GetAudience() []string }:
		for _, aud := range c.GetAudience() {
			if aud == audience {
				return true
			}
		}
	default:
		// If client doesn't implement GetAudience, allow any audience for now
		// In production, you might want to be more restrictive
		return true
	}
	return false
}

// determineTokenExchangeScope determines the scope for token exchange
func (h *TokenHandlers) determineTokenExchangeScope(originalScope, requestedScope string) string {
	if requestedScope == "" {
		return originalScope
	}

	// For token exchange, we can be more permissive, but still validate
	if h.isScopeSubset(requestedScope, originalScope) {
		return requestedScope
	}

	// Return original scope if requested scope is invalid
	return originalScope
}

// isScopeSubset checks if requestedScope is a subset of originalScope
func (h *TokenHandlers) isScopeSubset(requestedScope, originalScope string) bool {
	if requestedScope == "" {
		return true
	}

	originalScopes := strings.Split(originalScope, " ")
	requestedScopes := strings.Split(requestedScope, " ")

	for _, reqScope := range requestedScopes {
		if reqScope == "" {
			continue
		}
		found := false
		for _, origScope := range originalScopes {
			if origScope == reqScope {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
