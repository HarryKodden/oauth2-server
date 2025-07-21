package utils

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ValidateClientID checks if the provided client ID is valid
func ValidateClientID(clientID string) error {
	if clientID == "" {
		return errors.New("client ID cannot be empty")
	}
	if len(clientID) < 3 {
		return errors.New("client ID too short")
	}
	return nil
}

// ValidateClientSecret checks if the provided client secret is valid
func ValidateClientSecret(clientSecret string) error {
	if clientSecret == "" {
		return errors.New("client secret cannot be empty")
	}
	if len(clientSecret) < 8 {
		return errors.New("client secret too short")
	}
	return nil
}

// ValidateRedirectURI validates if a redirect URI is properly formatted
func ValidateRedirectURI(redirectURI string) bool {
	if redirectURI == "" {
		return false
	}

	// Allow special OAuth 2.0 URIs
	if redirectURI == "urn:ietf:wg:oauth:2.0:oob" {
		return true
	}

	// Parse and validate the URL
	parsedURL, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}

	// Must have a scheme
	if parsedURL.Scheme == "" {
		return false
	}

	// Must be HTTP or HTTPS (or custom schemes for mobile apps)
	validSchemes := []string{"http", "https", "com.example.app", "myapp"}
	schemeValid := false
	for _, scheme := range validSchemes {
		if parsedURL.Scheme == scheme {
			schemeValid = true
			break
		}
	}

	return schemeValid
}

// ValidateGrantType checks if a grant type is supported
func ValidateGrantType(grantType string) bool {
	supportedGrantTypes := []string{
		"authorization_code",
		"client_credentials",
		"refresh_token",
		"urn:ietf:params:oauth:grant-type:device_code",
		"urn:ietf:params:oauth:grant-type:token-exchange",
	}

	return contains(supportedGrantTypes, grantType)
}

// ValidateResponseType checks if a response type is supported
func ValidateResponseType(responseType string) bool {
	supportedResponseTypes := []string{
		"code",
		"token",
		"id_token",
		"code token",
		"code id_token",
		"token id_token",
		"code token id_token",
	}

	return contains(supportedResponseTypes, responseType)
}

// ValidateScope checks if requested scopes are valid against allowed scopes
func ValidateScope(requestedScope, allowedScopes string) bool {
	if requestedScope == "" {
		return true
	}

	requestedList := SplitScopes(requestedScope)
	allowedList := SplitScopes(allowedScopes)

	for _, requested := range requestedList {
		found := false
		for _, allowed := range allowedList {
			if requested == allowed {
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
    
    return nil
}


// ValidateUserCode checks if the user code format is valid
func ValidateUserCode(userCode string) error {
	if userCode == "" {
		return errors.New("user code cannot be empty")
	}

	// Remove hyphens and spaces for validation
	cleanCode := strings.ReplaceAll(strings.ReplaceAll(userCode, "-", ""), " ", "")
	
	// User codes should be 6-8 characters, alphanumeric
	re := regexp.MustCompile(`^[A-Z0-9]{6,8}$`)
	if !re.MatchString(cleanCode) {
		return errors.New("invalid user code format")
	}
	return nil
}

// ValidateEmail checks if the email format is valid
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

// ValidateCodeChallenge validates PKCE code challenge
func ValidateCodeChallenge(challenge string) error {
	if challenge == "" {
		return errors.New("code challenge cannot be empty")
	}

	// Code challenge should be 43-128 characters, base64url encoded
	if len(challenge) < 43 || len(challenge) > 128 {
		return errors.New("code challenge length must be between 43 and 128 characters")
	}

	// Check for valid base64url characters
	re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !re.MatchString(challenge) {
		return errors.New("code challenge contains invalid characters")
	}

	return nil
}

// ValidateCodeChallengeMethod validates PKCE code challenge method
func ValidateCodeChallengeMethod(method string) bool {
	return method == "plain" || method == "S256"
}

// contains is a helper function to check if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}