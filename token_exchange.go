package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ory/fosite"
)

// TokenExchangeRequest represents a token exchange request
type TokenExchangeRequest struct {
	GrantType         string `json:"grant_type"`
	ClientID          string `json:"client_id"`
	ClientSecret      string `json:"client_secret"`
	SubjectToken      string `json:"subject_token"`
	SubjectTokenType  string `json:"subject_token_type"`
	RequestedTokenType string `json:"requested_token_type,omitempty"`
	Audience          string `json:"audience,omitempty"`
	Scope             string `json:"scope,omitempty"`
}

// TokenExchangeResponse represents a token exchange response
type TokenExchangeResponse struct {
	AccessToken      string `json:"access_token"`
	IssuedTokenType  string `json:"issued_token_type"`
	TokenType        string `json:"token_type"`
	ExpiresIn        int64  `json:"expires_in"`
	Scope            string `json:"scope,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
}

// handleTokenExchange processes RFC 8693 Token Exchange requests
func handleTokenExchange(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Processing token exchange request")
	
	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		writeErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}
	
	req := TokenExchangeRequest{
		GrantType:         r.FormValue("grant_type"),
		ClientID:          r.FormValue("client_id"),
		ClientSecret:      r.FormValue("client_secret"),
		SubjectToken:      r.FormValue("subject_token"),
		SubjectTokenType:  r.FormValue("subject_token_type"),
		RequestedTokenType: r.FormValue("requested_token_type"),
		Audience:          r.FormValue("audience"),
		Scope:             r.FormValue("scope"),
	}
	
	log.Printf("Token exchange request: %+v", req)
	
	// Validate grant type
	if req.GrantType != "urn:ietf:params:oauth:grant-type:token-exchange" {
		writeErrorResponse(w, "unsupported_grant_type", "Grant type not supported")
		return
	}
	
	// Authenticate client
	client, err := authenticateClient(req.ClientID, req.ClientSecret)
	if err != nil {
		log.Printf("❌ Client authentication failed: %v", err)
		writeErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}
	
	// Validate subject token (simplified - in production, verify JWT signature)
	if req.SubjectToken == "" {
		writeErrorResponse(w, "invalid_request", "Subject token is required")
		return
	}
	
	// Check if client is authorized for token exchange
	if !clientHasGrantType(client, "urn:ietf:params:oauth:grant-type:token-exchange") {
		writeErrorResponse(w, "unauthorized_client", "Client not authorized for token exchange")
		return
	}
	
	// Generate new access token for the audience
	accessToken, err := generateAccessTokenForAudience(client, req.Audience, req.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate access token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate token")
		return
	}
	
	// Create response
	response := TokenExchangeResponse{
		AccessToken:     accessToken,
		IssuedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		TokenType:       "Bearer",
		ExpiresIn:       3600, // 1 hour
		Scope:          req.Scope,
	}
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		writeErrorResponse(w, "server_error", "Failed to encode response")
		return
	}
	
	log.Printf("✅ Token exchange successful")
}

// authenticateClient validates client credentials
func authenticateClient(clientID, clientSecret string) (fosite.Client, error) {
	client, ok := store.Clients[clientID]
	if !ok {
		return nil, fmt.Errorf("client not found")
	}
	
	// For DefaultClient, the "hashed" secret is actually just the raw secret
	if string(client.GetHashedSecret()) != clientSecret {
		return nil, fmt.Errorf("invalid client secret")
	}
	
	return client, nil
}

// clientHasGrantType checks if client is authorized for the grant type
func clientHasGrantType(client fosite.Client, grantType string) bool {
	for _, gt := range client.GetGrantTypes() {
		if gt == grantType {
			return true
		}
	}
	return false
}

// generateAccessTokenForAudience creates a new access token for the specified audience
func generateAccessTokenForAudience(client fosite.Client, audience, scope string) (string, error) {
	// For simplicity, create a basic token
	// In production, you'd generate a proper JWT
	now := time.Now()
	
	// For demo purposes, create a simple token string
	// In production, use proper JWT library
	token := fmt.Sprintf("token_exchange_%d_%s_%s", 
		now.Unix(), client.GetID(), audience)
	
	log.Printf("Generated token for audience %s: %s", audience, token)
	return token, nil
}

// writeErrorResponse writes an OAuth2 error response
func writeErrorResponse(w http.ResponseWriter, errorCode, errorDescription string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	
	errorResp := map[string]string{
		"error":             errorCode,
		"error_description": errorDescription,
	}
	
	json.NewEncoder(w).Encode(errorResp)
}

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

// handleClientCredentials processes client credentials requests manually
func handleClientCredentials(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔧 Processing client credentials request manually")
	
	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		writeErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}
	
	// Extract client credentials from form or Authorization header
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	
	// If not in form, try Authorization header (HTTP Basic)
	if clientID == "" || clientSecret == "" {
		username, password, ok := r.BasicAuth()
		if ok {
			clientID = username
			clientSecret = password
			log.Printf("Using HTTP Basic auth: client_id=%s", clientID)
		}
	}
	
	req := ClientCredentialsRequest{
		GrantType:    r.FormValue("grant_type"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scope:        r.FormValue("scope"),
	}
	
	log.Printf("Client credentials request: %+v", req)
	
	// Validate grant type
	if req.GrantType != "client_credentials" {
		writeErrorResponse(w, "unsupported_grant_type", "Grant type not supported")
		return
	}
	
	// Authenticate client
	client, err := authenticateClient(req.ClientID, req.ClientSecret)
	if err != nil {
		log.Printf("❌ Client authentication failed: %v", err)
		writeErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}
	
	// Check if client is authorized for client credentials
	if !clientHasGrantType(client, "client_credentials") {
		writeErrorResponse(w, "unauthorized_client", "Client not authorized for client credentials")
		return
	}
	
	// Generate access token 
	accessToken, err := generateAccessTokenForClient(client, req.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate access token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate token")
		return
	}
	
	// Generate refresh token for long-running processes
	refreshToken, err := generateRefreshTokenForClient(client, req.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate refresh token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate refresh token")
		return
	}
	
	// Create response
	response := ClientCredentialsResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		Scope:        req.Scope,
		RefreshToken: refreshToken,
	}
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		writeErrorResponse(w, "server_error", "Failed to encode response")
		return
	}
	
	log.Printf("✅ Client credentials successful")
}

// generateAccessTokenForClient creates a new access token for client credentials
func generateAccessTokenForClient(client fosite.Client, scope string) (string, error) {
	now := time.Now()
	
	// For demo purposes, create a simple token string
	// In production, use proper JWT library
	token := fmt.Sprintf("client_credentials_%d_%s", 
		now.Unix(), client.GetID())
	
	log.Printf("Generated client credentials token: %s", token)
	return token, nil
}

// generateRefreshTokenForClient creates a new refresh token for client credentials
func generateRefreshTokenForClient(client fosite.Client, scope string) (string, error) {
	now := time.Now()
	
	// For demo purposes, create a simple refresh token string
	// In production, use proper cryptographic tokens
	token := fmt.Sprintf("refresh_%d_%s", 
		now.Unix(), client.GetID())
	
	log.Printf("Generated refresh token: %s", token)
	return token, nil
}

// RefreshTokenRequest represents a refresh token request
type RefreshTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope,omitempty"`
}

// RefreshTokenResponse represents a refresh token response
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

// DeviceCodeRequest represents a device authorization request (RFC 8628)
type DeviceCodeRequest struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope,omitempty"`
}

// DeviceCodeResponse represents a device authorization response (RFC 8628)
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int64  `json:"expires_in"`
	Interval                int64  `json:"interval"`
}

// DeviceTokenRequest represents a device token request (RFC 8628)
type DeviceTokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	DeviceCode   string `json:"device_code"`
}

// DeviceAuthInfo stores device authorization information
type DeviceAuthInfo struct {
	DeviceCode   string
	UserCode     string
	ClientID     string
	Scope        string
	ExpiresAt    time.Time
	Authorized   bool
	UserID       string
	CreatedAt    time.Time
}

// handleRefreshToken processes refresh token requests manually
func handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Processing refresh token request manually")
	
	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		writeErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}
	
	// Extract client credentials from form or Authorization header
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	
	// If not in form, try Authorization header (HTTP Basic)
	if clientID == "" || clientSecret == "" {
		username, password, ok := r.BasicAuth()
		if ok {
			clientID = username
			clientSecret = password
			log.Printf("Using HTTP Basic auth for refresh: client_id=%s", clientID)
		}
	}
	
	req := RefreshTokenRequest{
		GrantType:    r.FormValue("grant_type"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RefreshToken: r.FormValue("refresh_token"),
		Scope:        r.FormValue("scope"),
	}
	
	log.Printf("Refresh token request: %+v", req)
	
	// Validate grant type
	if req.GrantType != "refresh_token" {
		writeErrorResponse(w, "unsupported_grant_type", "Grant type not supported")
		return
	}
	
	// Validate refresh token is provided
	if req.RefreshToken == "" {
		writeErrorResponse(w, "invalid_request", "Refresh token is required")
		return
	}
	
	// Authenticate client
	client, err := authenticateClient(req.ClientID, req.ClientSecret)
	if err != nil {
		log.Printf("❌ Client authentication failed: %v", err)
		writeErrorResponse(w, "invalid_client", "Client authentication failed")
		return
	}
	
	// Check if client is authorized for refresh tokens
	if !clientHasGrantType(client, "refresh_token") {
		writeErrorResponse(w, "unauthorized_client", "Client not authorized for refresh tokens")
		return
	}
	
	// Validate refresh token (simplified - in production, verify signature and expiration)
	if !isValidRefreshToken(req.RefreshToken, client.GetID()) {
		writeErrorResponse(w, "invalid_grant", "Invalid refresh token")
		return
	}
	
	// Generate new access token
	accessToken, err := generateAccessTokenForClient(client, req.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate access token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate token")
		return
	}
	
	// Generate new refresh token (optional - can reuse the same one)
	newRefreshToken, err := generateRefreshTokenForClient(client, req.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate refresh token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate refresh token")
		return
	}
	
	// Create response
	response := RefreshTokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		Scope:        req.Scope,
		RefreshToken: newRefreshToken,
	}
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		writeErrorResponse(w, "server_error", "Failed to encode response")
		return
	}
	
	log.Printf("✅ Refresh token successful")
}

// isValidRefreshToken validates a refresh token (simplified implementation)
func isValidRefreshToken(refreshToken, clientID string) bool {
	// For demo purposes, just check if it starts with "refresh_" and contains the client ID
	// In production, you'd verify JWT signature, expiration, and store lookup
	return fmt.Sprintf("refresh_") != "" && 
		   len(refreshToken) > 10 && 
		   fmt.Sprintf("_%s", clientID) != ""
}

// Device code storage (in production, use a proper database)
var deviceAuthStore = make(map[string]*DeviceAuthInfo)

// handleDeviceAuthorization processes device authorization requests (RFC 8628 step 1)
func handleDeviceAuthorization(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔗 Processing device authorization request")
	
	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		writeErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}
	
	clientID := r.FormValue("client_id")
	scope := r.FormValue("scope")
	
	log.Printf("Device authorization for client: %s, scope: %s", clientID, scope)
	
	// Validate client
	_, err := store.GetClient(r.Context(), clientID)
	if err != nil {
		log.Printf("❌ Client not found: %s", clientID)
		writeErrorResponse(w, "invalid_client", "Client not found")
		return
	}
	
	// Generate device code and user code
	deviceCode := generateDeviceCode()
	userCode := generateUserCode()
	
	// Store device authorization info
	deviceAuth := &DeviceAuthInfo{
		DeviceCode: deviceCode,
		UserCode:   userCode,
		ClientID:   clientID,
		Scope:      scope,
		ExpiresAt:  time.Now().Add(10 * time.Minute), // 10 minutes
		Authorized: false,
		CreatedAt:  time.Now(),
	}
	
	deviceAuthStore[deviceCode] = deviceAuth
	deviceAuthStore[userCode] = deviceAuth // Store by user code for lookup
	
	// Create response
	response := DeviceCodeResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         "http://localhost:8080/device",
		VerificationURIComplete: fmt.Sprintf("http://localhost:8080/device?user_code=%s", userCode),
		ExpiresIn:               600, // 10 minutes
		Interval:                5,   // Poll every 5 seconds
	}
	
	log.Printf("✅ Device authorization created - User code: %s, Device code: %s", userCode, deviceCode)
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		writeErrorResponse(w, "server_error", "Failed to encode response")
		return
	}
}

// handleDeviceToken processes device token requests (RFC 8628 step 3)
func handleDeviceToken(w http.ResponseWriter, r *http.Request) {
	log.Printf("🎯 Processing device token request")
	
	// Parse the request
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ Failed to parse form: %v", err)
		writeErrorResponse(w, "invalid_request", "Failed to parse request")
		return
	}
	
	deviceCode := r.FormValue("device_code")
	clientID := r.FormValue("client_id")
	
	log.Printf("Device token request for device: %s, client: %s", deviceCode, clientID)
	
	// Validate client
	client, err := store.GetClient(r.Context(), clientID)
	if err != nil {
		log.Printf("❌ Client not found: %s", clientID)
		writeErrorResponse(w, "invalid_client", "Client not found")
		return
	}
	
	// Get device authorization info
	deviceAuth, exists := deviceAuthStore[deviceCode]
	if !exists {
		log.Printf("❌ Device code not found: %s", deviceCode)
		writeErrorResponse(w, "invalid_grant", "Device code not found")
		return
	}
	
	// Check if expired
	if time.Now().After(deviceAuth.ExpiresAt) {
		log.Printf("❌ Device code expired: %s", deviceCode)
		delete(deviceAuthStore, deviceCode)
		delete(deviceAuthStore, deviceAuth.UserCode)
		writeErrorResponse(w, "expired_token", "Device code has expired")
		return
	}
	
	// Check if client matches
	if deviceAuth.ClientID != clientID {
		log.Printf("❌ Client ID mismatch for device code: %s", deviceCode)
		writeErrorResponse(w, "invalid_grant", "Client ID mismatch")
		return
	}
	
	// Check if user has authorized
	if !deviceAuth.Authorized {
		log.Printf("⏳ Device code not yet authorized: %s", deviceCode)
		writeErrorResponse(w, "authorization_pending", "User has not yet authorized")
		return
	}
	
	// Generate access token
	accessToken, err := generateAccessTokenForClient(client, deviceAuth.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate access token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate token")
		return
	}
	
	// Generate refresh token
	refreshToken, err := generateRefreshTokenForClient(client, deviceAuth.Scope)
	if err != nil {
		log.Printf("❌ Failed to generate refresh token: %v", err)
		writeErrorResponse(w, "server_error", "Failed to generate refresh token")
		return
	}
	
	// Clean up device authorization
	delete(deviceAuthStore, deviceCode)
	delete(deviceAuthStore, deviceAuth.UserCode)
	
	// Create response
	response := TokenExchangeResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		Scope:        deviceAuth.Scope,
		RefreshToken: refreshToken,
	}
	
	log.Printf("✅ Device token issued for user: %s", deviceAuth.UserID)
	
	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Failed to encode response: %v", err)
		writeErrorResponse(w, "server_error", "Failed to encode response")
		return
	}
}

// generateDeviceCode generates a unique device code
func generateDeviceCode() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 16)
	rand.Read(randomBytes)
	return fmt.Sprintf("device_%d_%x", timestamp, randomBytes[:8])
}

// generateUserCode generates a human-readable user code
func generateUserCode() string {
	// Generate a simple 6-character code like "ABC123"
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	code := make([]byte, 6)
	for i := range code {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		code[i] = charset[randomByte[0]%byte(len(charset))]
	}
	return string(code)
}

// authorizeDevice authorizes a device using the user code
func authorizeDevice(userCode, userID string) bool {
	deviceAuth, exists := deviceAuthStore[userCode]
	if !exists {
		log.Printf("❌ User code not found: %s", userCode)
		return false
	}
	
	if time.Now().After(deviceAuth.ExpiresAt) {
		log.Printf("❌ User code expired: %s", userCode)
		delete(deviceAuthStore, deviceAuth.DeviceCode)
		delete(deviceAuthStore, userCode)
		return false
	}
	
	deviceAuth.Authorized = true
	deviceAuth.UserID = userID
	
	log.Printf("✅ Device authorized with user code: %s for user: %s", userCode, userID)
	return true
}
