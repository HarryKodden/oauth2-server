package store

import (
	"errors"
	"sync"
	"time"
)

// Token represents an access or refresh token
type Token struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// TokenInfo represents token information for validation
type TokenInfo struct {
	Token     string    `json:"token"`
	TokenType string    `json:"token_type"`
	ClientID  string    `json:"client_id"`
	UserID    string    `json:"user_id"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
	IssuedAt  time.Time `json:"iat"`
	Issuer    string    `json:"iss"`
	Audience  []string  `json:"aud"`
}

// TokenStore manages tokens
type TokenStore struct {
	tokens        map[string]*Token
	refreshTokens map[string]*Token // Separate storage for refresh tokens
	mutex         sync.RWMutex
}

// NewTokenStore creates a new token store
func NewTokenStore() *TokenStore {
	return &TokenStore{
		tokens:        make(map[string]*Token),
		refreshTokens: make(map[string]*Token),
	}
}

// StoreToken stores a token
func (s *TokenStore) StoreToken(token *Token) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if token.TokenType == "refresh" {
		s.refreshTokens[token.Token] = token
	} else {
		s.tokens[token.Token] = token
	}

	return nil
}

// StoreAccessToken stores an access token
func (s *TokenStore) StoreAccessToken(tokenString, clientID, userID string, scopes []string, expiresAt time.Time) error {
	token := &Token{
		Token:     tokenString,
		TokenType: "access",
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	return s.StoreToken(token)
}

// StoreRefreshToken stores a refresh token
func (s *TokenStore) StoreRefreshToken(tokenString, clientID, userID string, expiresAt time.Time) error {
	token := &Token{
		Token:     tokenString,
		TokenType: "refresh",
		ClientID:  clientID,
		UserID:    userID,
		Scopes:    []string{}, // Refresh tokens typically don't store scopes
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	return s.StoreToken(token)
}

// GetToken retrieves a token
func (s *TokenStore) GetToken(token string) (*Token, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check access tokens first
	if tokenData, exists := s.tokens[token]; exists {
		return tokenData, nil
	}

	// Check refresh tokens
	if tokenData, exists := s.refreshTokens[token]; exists {
		return tokenData, nil
	}

	return nil, errors.New("token not found")
}

// GetAccessToken retrieves an access token
func (s *TokenStore) GetAccessToken(token string) (*Token, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tokenData, exists := s.tokens[token]
	if !exists {
		return nil, errors.New("access token not found")
	}

	return tokenData, nil
}

// GetRefreshToken retrieves a refresh token
func (s *TokenStore) GetRefreshToken(token string) (*Token, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tokenData, exists := s.refreshTokens[token]
	if !exists {
		return nil, errors.New("refresh token not found")
	}

	return tokenData, nil
}

// ValidateRefreshToken validates a refresh token and returns token info
func (s *TokenStore) ValidateRefreshToken(token string) (*TokenInfo, error) {
	refreshToken, err := s.GetRefreshToken(token)
	if err != nil {
		return nil, err
	}

	if refreshToken.Revoked {
		return nil, errors.New("refresh token has been revoked")
	}

	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, errors.New("refresh token has expired")
	}

	// Convert to TokenInfo
	tokenInfo := &TokenInfo{
		Token:     refreshToken.Token,
		TokenType: refreshToken.TokenType,
		ClientID:  refreshToken.ClientID,
		UserID:    refreshToken.UserID,
		Scopes:    refreshToken.Scopes,
		ExpiresAt: refreshToken.ExpiresAt,
		Active:    true,
		IssuedAt:  refreshToken.CreatedAt,
		Issuer:    "oauth2-server",
		Audience:  []string{"api"},
	}

	return tokenInfo, nil
}

// ValidateAccessToken validates an access token and returns token info
func (s *TokenStore) ValidateAccessToken(token string) (*TokenInfo, error) {
	accessToken, err := s.GetAccessToken(token)
	if err != nil {
		return nil, err
	}

	if accessToken.Revoked {
		return nil, errors.New("access token has been revoked")
	}

	if time.Now().After(accessToken.ExpiresAt) {
		return nil, errors.New("access token has expired")
	}

	// Convert to TokenInfo
	tokenInfo := &TokenInfo{
		Token:     accessToken.Token,
		TokenType: accessToken.TokenType,
		ClientID:  accessToken.ClientID,
		UserID:    accessToken.UserID,
		Scopes:    accessToken.Scopes,
		ExpiresAt: accessToken.ExpiresAt,
		Active:    true,
		IssuedAt:  accessToken.CreatedAt,
		Issuer:    "oauth2-server",
		Audience:  []string{"api"},
	}

	return tokenInfo, nil
}

// RevokeToken marks a token as revoked
func (s *TokenStore) RevokeToken(token string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check access tokens first
	if tokenData, exists := s.tokens[token]; exists {
		tokenData.Revoked = true
		return nil
	}

	// Check refresh tokens
	if tokenData, exists := s.refreshTokens[token]; exists {
		tokenData.Revoked = true
		return nil
	}

	return errors.New("token not found")
}

// RevokeAccessToken revokes an access token
func (s *TokenStore) RevokeAccessToken(token string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tokenData, exists := s.tokens[token]
	if !exists {
		return errors.New("access token not found")
	}

	tokenData.Revoked = true
	return nil
}

// RevokeRefreshToken revokes a refresh token
func (s *TokenStore) RevokeRefreshToken(token string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	tokenData, exists := s.refreshTokens[token]
	if !exists {
		return errors.New("refresh token not found")
	}

	tokenData.Revoked = true
	return nil
}

// IsTokenValid checks if a token is valid (not expired or revoked)
func (s *TokenStore) IsTokenValid(token string) bool {
	tokenData, err := s.GetToken(token)
	if err != nil {
		return false
	}

	if tokenData.Revoked {
		return false
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return false
	}

	return true
}

// IsAccessTokenValid checks if an access token is valid
func (s *TokenStore) IsAccessTokenValid(token string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tokenData, exists := s.tokens[token]
	if !exists {
		return false
	}

	if tokenData.Revoked {
		return false
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return false
	}

	return true
}

// IsRefreshTokenValid checks if a refresh token is valid
func (s *TokenStore) IsRefreshTokenValid(token string) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tokenData, exists := s.refreshTokens[token]
	if !exists {
		return false
	}

	if tokenData.Revoked {
		return false
	}

	if time.Now().After(tokenData.ExpiresAt) {
		return false
	}

	return true
}

// CleanupExpiredTokens removes expired tokens
func (s *TokenStore) CleanupExpiredTokens() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	var expiredTokens []string
	var expiredRefreshTokens []string

	// Clean up access tokens
	for token, tokenData := range s.tokens {
		if now.After(tokenData.ExpiresAt) {
			expiredTokens = append(expiredTokens, token)
		}
	}

	// Clean up refresh tokens
	for token, tokenData := range s.refreshTokens {
		if now.After(tokenData.ExpiresAt) {
			expiredRefreshTokens = append(expiredRefreshTokens, token)
		}
	}

	// Delete expired tokens
	for _, token := range expiredTokens {
		delete(s.tokens, token)
	}

	for _, token := range expiredRefreshTokens {
		delete(s.refreshTokens, token)
	}

	totalCleaned := len(expiredTokens) + len(expiredRefreshTokens)
	return totalCleaned
}

// GetTokensByUser retrieves all tokens for a specific user
func (s *TokenStore) GetTokensByUser(userID string) ([]*Token, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var userTokens []*Token

	// Check access tokens
	for _, token := range s.tokens {
		if token.UserID == userID {
			userTokens = append(userTokens, token)
		}
	}

	// Check refresh tokens
	for _, token := range s.refreshTokens {
		if token.UserID == userID {
			userTokens = append(userTokens, token)
		}
	}

	return userTokens, nil
}

// GetTokensByClient retrieves all tokens for a specific client
func (s *TokenStore) GetTokensByClient(clientID string) ([]*Token, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var clientTokens []*Token

	// Check access tokens
	for _, token := range s.tokens {
		if token.ClientID == clientID {
			clientTokens = append(clientTokens, token)
		}
	}

	// Check refresh tokens
	for _, token := range s.refreshTokens {
		if token.ClientID == clientID {
			clientTokens = append(clientTokens, token)
		}
	}

	return clientTokens, nil
}

// GetStats returns statistics about stored tokens
func (s *TokenStore) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	now := time.Now()
	var activeAccess, expiredAccess, revokedAccess int
	var activeRefresh, expiredRefresh, revokedRefresh int

	// Count access token stats
	for _, token := range s.tokens {
		if token.Revoked {
			revokedAccess++
		} else if now.After(token.ExpiresAt) {
			expiredAccess++
		} else {
			activeAccess++
		}
	}

	// Count refresh token stats
	for _, token := range s.refreshTokens {
		if token.Revoked {
			revokedRefresh++
		} else if now.After(token.ExpiresAt) {
			expiredRefresh++
		} else {
			activeRefresh++
		}
	}

	return map[string]interface{}{
		"access_tokens": map[string]int{
			"total":   len(s.tokens),
			"active":  activeAccess,
			"expired": expiredAccess,
			"revoked": revokedAccess,
		},
		"refresh_tokens": map[string]int{
			"total":   len(s.refreshTokens),
			"active":  activeRefresh,
			"expired": expiredRefresh,
			"revoked": revokedRefresh,
		},
	}
}