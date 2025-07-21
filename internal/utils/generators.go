package utils

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"
)

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// GenerateUserCode generates a user-friendly code for device flow
func GenerateUserCode() string {
	// Generate 8-character code using base32 (excludes confusing characters)
	bytes := make([]byte, 5)
	rand.Read(bytes)
	code := base32.StdEncoding.EncodeToString(bytes)
	code = strings.TrimRight(code, "=") // Remove padding

	// Format as XXXX-XXXX for better readability
	if len(code) >= 8 {
		return fmt.Sprintf("%s-%s", code[:4], code[4:8])
	}
	return code
}

// GenerateAuthorizationCode generates an authorization code
func GenerateAuthorizationCode() string {
	return GenerateRandomString(32)
}

// GenerateAccessToken generates an access token
func GenerateAccessToken() string {
	return GenerateRandomString(32)
}

// GenerateRefreshToken generates a refresh token
func GenerateRefreshToken() string {
	return GenerateRandomString(32)
}

// GenerateClientID generates a client ID
func GenerateClientID() string {
	return GenerateRandomString(20)
}

// GenerateClientSecret generates a client secret
func GenerateClientSecret() string {
	return GenerateRandomString(40)
}

// GenerateRegistrationAccessToken generates a registration access token
func GenerateRegistrationAccessToken() string {
	return GenerateRandomString(32)
}
