package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateRandomString generates a cryptographically secure random string
func GenerateRandomString(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateUserCode generates a user-friendly device verification code
func GenerateUserCode() string {
	// Generate a shorter, more user-friendly code
	code, _ := GenerateRandomString(8)
	return strings.ToUpper(code)
}

// GenerateState generates a state parameter for OAuth2 flows
func GenerateState() string {
	state, _ := GenerateRandomString(32)
	return state
}

// GenerateNonce generates a nonce for OpenID Connect
func GenerateNonce() string {
	nonce, _ := GenerateRandomString(32)
	return nonce
}

// GenerateAccessToken generates an access token
func GenerateAccessToken() string {
	token, _ := GenerateRandomString(32)
	return token
}

// GenerateRefreshToken generates a refresh token
func GenerateRefreshToken() string {
	token, _ := GenerateRandomString(32)
	return token
}

// GenerateClientID generates a unique client ID
func GenerateClientID() string {
	id, _ := GenerateRandomString(16)
	return "client_" + id
}

// GenerateClientSecret generates a secure client secret
func GenerateClientSecret() string {
	secret, _ := GenerateRandomString(32)
	return secret
}

// GenerateAuthCode generates an authorization code
func GenerateAuthCode() string {
	code, _ := GenerateRandomString(32)
	return code
}

// GenerateDeviceCode generates a device code
func GenerateDeviceCode() string {
	code, _ := GenerateRandomString(32)
	return code
}

// GenerateCodeChallenge generates a PKCE code challenge
func GenerateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// GenerateCodeVerifier generates a PKCE code verifier
func GenerateCodeVerifier() string {
	verifier, _ := GenerateRandomString(128)
	return verifier
}
