package auth

import (
    "errors"
    "strings"
)

// ValidateRegistrationAccessToken validates a registration access token
func ValidateRegistrationAccessToken(token string) error {
    if token == "" {
        return errors.New("registration access token is required")
    }

    // Basic validation - check token format and length
    if len(token) < 20 {
        return errors.New("invalid registration access token format")
    }

    // In a real implementation, you would:
    // 1. Parse and validate the token
    // 2. Check token expiration
    // 3. Verify token signature
    // 4. Check token against registration token store

    return nil
}

// ExtractRegistrationToken extracts registration token from Authorization header
func ExtractRegistrationToken(authHeader string) (string, error) {
    parts := strings.SplitN(authHeader, " ", 2)
    if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
        return "", errors.New("invalid authorization header format")
    }
    return parts[1], nil
}