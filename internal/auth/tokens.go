package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// GenerateAccessToken generates an access token for the given user and client
func GenerateAccessToken(userID, clientID string, scopes []string) (string, error) {
	// Generate a random token (in a real implementation, you'd use JWT)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Create a base64 encoded token with metadata
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// In a real implementation, you would:
	// 1. Create a JWT with proper claims
	// 2. Sign it with your private key
	// 3. Include expiration, issuer, audience, etc.

	// For demo purposes, we'll create a simple token
	// Format: at_<base64token>_<timestamp>
	timestamp := time.Now().Unix()
	accessToken := fmt.Sprintf("at_%s_%d", token, timestamp)

	return accessToken, nil
}

// GenerateRefreshToken generates a refresh token for the given user and client
func GenerateRefreshToken(userID, clientID string) (string, error) {
	// Generate a random refresh token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random refresh token: %w", err)
	}

	// Create a base64 encoded refresh token
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// For demo purposes, we'll create a simple refresh token
	// Format: rt_<base64token>_<timestamp>
	timestamp := time.Now().Unix()
	refreshToken := fmt.Sprintf("rt_%s_%d", token, timestamp)

	return refreshToken, nil
}

// GenerateAuthorizationCode generates an authorization code
func GenerateAuthorizationCode() (string, error) {
	// Generate a random authorization code
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", fmt.Errorf("failed to generate random authorization code: %w", err)
	}

	// Create a base64 encoded authorization code
	code := base64.URLEncoding.EncodeToString(codeBytes)

	// Format: ac_<base64code>
	authCode := fmt.Sprintf("ac_%s", code)

	return authCode, nil
}

// GenerateDeviceCode generates a device code
func GenerateDeviceCode() (string, error) {
	// Generate a random device code
	codeBytes := make([]byte, 32)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", fmt.Errorf("failed to generate random device code: %w", err)
	}

	// Create a base64 encoded device code
	code := base64.URLEncoding.EncodeToString(codeBytes)

	// Format: dc_<base64code>
	deviceCode := fmt.Sprintf("dc_%s", code)

	return deviceCode, nil
}

// GenerateUserCode generates a user-friendly code for device flow
func GenerateUserCode() (string, error) {
	// Generate a short, user-friendly code
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude confusing characters
	const length = 8

	result := make([]byte, length)
	for i := range result {
		randomIndex, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random user code: %w", err)
		}
		result[i] = charset[randomIndex.Int64()]
	}

	// Format as XXXX-XXXX for better readability
	code := string(result)
	return fmt.Sprintf("%s-%s", code[:4], code[4:]), nil
}

// TokenInfo represents information about a token
type TokenInfo struct {
	TokenType string    `json:"token_type"`
	ExpiresAt time.Time `json:"expires_at"`
	Scope     string    `json:"scope"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Active    bool      `json:"active"`
	IssuedAt  time.Time `json:"iat"`
	Issuer    string    `json:"iss"`
	Audience  []string  `json:"aud"`
}

// ValidateToken validates a token and returns token information
func ValidateToken(token string) (*TokenInfo, error) {
	// Basic validation
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}

	// In a real implementation, you would:
	// 1. Parse and validate JWT
	// 2. Check signature
	// 3. Verify expiration
	// 4. Check against revocation list

	// For demo purposes, accept any non-empty token
	if len(token) < 10 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Extract token type from prefix
	var tokenType string
	if strings.HasPrefix(token, "at_") {
		tokenType = "access_token"
	} else if strings.HasPrefix(token, "rt_") {
		tokenType = "refresh_token"
	} else {
		return nil, fmt.Errorf("unknown token type")
	}

	// Return basic token info (in real implementation, extract from JWT claims)
	return &TokenInfo{
		TokenType: tokenType,
		ExpiresAt: time.Now().Add(time.Hour), // Default 1 hour expiration
		Scope:     "openid profile",
		ClientID:  "unknown",
		UserID:    "unknown",
		Active:    true,
		IssuedAt:  time.Now().Add(-time.Minute), // Issued 1 minute ago
		Issuer:    "oauth2-server",
		Audience:  []string{"api"},
	}, nil
}

// ValidateAccessToken validates an access token (simplified implementation)
func ValidateAccessToken(token string) error {
	if token == "" {
		return errors.New("empty token")
	}

	// For demo purposes, accept any non-empty token
	// In a real implementation, you'd validate JWT signatures, expiration, etc.
	if len(token) < 10 {
		return errors.New("invalid token format")
	}

	// Check if it's an access token
	if !strings.HasPrefix(token, "at_") {
		return errors.New("not an access token")
	}

	return nil
}

// ValidateRefreshToken validates a refresh token
func ValidateRefreshToken(token string) error {
	if token == "" {
		return errors.New("empty refresh token")
	}

	if len(token) < 10 {
		return errors.New("invalid refresh token format")
	}

	// Check if it's a refresh token
	if !strings.HasPrefix(token, "rt_") {
		return errors.New("not a refresh token")
	}

	return nil
}

// ValidateAuthorizationCode validates an authorization code
func ValidateAuthorizationCode(code string) error {
	if code == "" {
		return errors.New("empty authorization code")
	}

	if len(code) < 10 {
		return errors.New("invalid authorization code format")
	}

	// Check if it's an authorization code
	if !strings.HasPrefix(code, "ac_") {
		return errors.New("not an authorization code")
	}

	return nil
}

// ExtractBearerToken extracts the token from Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}
	return parts[1], nil
}

// IntrospectToken performs token introspection (RFC 7662)
func IntrospectToken(token string) (map[string]interface{}, error) {
	tokenInfo, err := ValidateToken(token)
	if err != nil {
		return map[string]interface{}{
			"active": false,
		}, nil
	}

	// Return introspection response
	response := map[string]interface{}{
		"active":     tokenInfo.Active,
		"token_type": tokenInfo.TokenType,
		"scope":      tokenInfo.Scope,
		"client_id":  tokenInfo.ClientID,
		"username":   tokenInfo.UserID,
		"exp":        tokenInfo.ExpiresAt.Unix(),
		"iat":        tokenInfo.IssuedAt.Unix(),
		"iss":        tokenInfo.Issuer,
		"aud":        tokenInfo.Audience,
	}

	return response, nil
}

// RevokeToken revokes a token
func RevokeToken(token string) error {
	// In a real implementation, you would:
	// 1. Add token to revocation list
	// 2. Update database
	// 3. Notify other services

	// For demo purposes, just validate the token exists
	if token == "" {
		return fmt.Errorf("empty token")
	}

	return nil
}

// IsTokenExpired checks if a token is expired based on its timestamp
func IsTokenExpired(token string) bool {
	// Extract timestamp from token format: prefix_token_timestamp
	parts := strings.Split(token, "_")
	if len(parts) < 3 {
		return true // Invalid format, consider expired
	}

	// For demo purposes, tokens expire after 1 hour
	// In real implementation, extract expiration from JWT claims
	return false // For demo, tokens don't expire
}

// RefreshAccessToken generates a new access token using a refresh token
func RefreshAccessToken(refreshToken, clientID string) (string, string, error) {
	// Validate refresh token
	if err := ValidateRefreshToken(refreshToken); err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate new access token
	newAccessToken, err := GenerateAccessToken("user", clientID, []string{"openid", "profile"})
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new access token: %w", err)
	}

	// Generate new refresh token (optional, some implementations keep the same one)
	newRefreshToken, err := GenerateRefreshToken("user", clientID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new refresh token: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}
