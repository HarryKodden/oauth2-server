package utils

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ErrInvalidRedirectURI represents an invalid redirect URI error
var ErrInvalidRedirectURI = errors.New("invalid redirect URI")

// ValidateRedirectURI validates if a redirect URI is valid
func ValidateRedirectURI(uri string) error {
	if uri == "" {
		return errors.New("redirect URI must not be empty")
	}

	parsedURI, err := url.Parse(uri)
	if err != nil {
		return errors.New("redirect URI must be a valid URI")
	}

	// Check for custom schemes (for mobile apps)
	if parsedURI.Scheme != "http" && parsedURI.Scheme != "https" && !strings.Contains(parsedURI.Scheme, ".") {
		// Allow custom schemes for mobile apps (e.g., "com.example.app://oauth")
		return nil
	}

	// For HTTP/HTTPS URIs, validate more strictly
	if parsedURI.Scheme != "http" && parsedURI.Scheme != "https" {
		return errors.New("redirect URI must use http or https scheme")
	}

	if parsedURI.Fragment != "" {
		return errors.New("redirect URI must not contain a fragment")
	}

	return nil
}

// ValidateGrantType validates if a grant type is supported
func ValidateGrantType(grantType string) bool {
	supportedTypes := []string{
		"authorization_code",
		"client_credentials",
		"refresh_token",
		"urn:ietf:params:oauth:grant-type:device_code",
		"urn:ietf:params:oauth:grant-type:token-exchange",
	}

	for _, supported := range supportedTypes {
		if grantType == supported {
			return true
		}
	}
	return false
}

// ValidateResponseType validates if a response type is supported
func ValidateResponseType(responseType string) bool {
	supportedTypes := []string{"code", "token", "id_token"}

	for _, supported := range supportedTypes {
		if responseType == supported {
			return true
		}
	}
	return false
}

// IsValidRedirectURI validates a redirect URI (simple check)
func IsValidRedirectURI(uri string) bool {
	if uri == "" {
		return false
	}

	// Basic URL validation - in a real implementation, you'd be more thorough
	return strings.HasPrefix(uri, "http://") ||
		strings.HasPrefix(uri, "https://") ||
		strings.Contains(uri, "://") // Allow custom schemes for mobile apps
}

// ValidateScope validates if a scope is valid
func ValidateScope(scope string) bool {
	if scope == "" {
		return false
	}

	// Basic scope validation - alphanumeric plus some special characters
	for _, char := range scope {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-' || char == '.' || char == ':') {
			return false
		}
	}
	return true
}

// ValidateClientID validates if a client ID is valid format
func ValidateClientID(clientID string) bool {
	if clientID == "" {
		return false
	}

	// Client ID should be alphanumeric with underscores and hyphens
	for _, char := range clientID {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_' || char == '-') {
			return false
		}
	}
	return len(clientID) >= 3 && len(clientID) <= 128
}

// ValidateUserCode validates a user code for device flow
func ValidateUserCode(userCode string) error {
	if userCode == "" {
		return errors.New("user code cannot be empty")
	}

	// Remove hyphens and spaces for validation (common formatting)
	cleanCode := strings.ReplaceAll(strings.ReplaceAll(userCode, "-", ""), " ", "")

	// User codes should be 6-8 characters, alphanumeric
	if len(cleanCode) < 6 || len(cleanCode) > 8 {
		return errors.New("user code must be 6-8 characters long")
	}

	// Check for valid characters (alphanumeric, uppercase)
	re := regexp.MustCompile(`^[A-Z0-9]+$`)
	if !re.MatchString(cleanCode) {
		return errors.New("user code must contain only uppercase letters and numbers")
	}

	return nil
}

// ValidateEmail validates if an email address is valid
func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email cannot be empty")
	}

	// Basic email validation using regex
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return errors.New("invalid email format")
	}

	return nil
}

// ValidateCodeChallenge validates a PKCE code challenge
func ValidateCodeChallenge(challenge string) error {
	if challenge == "" {
		return errors.New("code challenge cannot be empty")
	}

	// Code challenge should be 43-128 characters for base64url encoding
	if len(challenge) < 43 || len(challenge) > 128 {
		return errors.New("code challenge must be between 43 and 128 characters")
	}

	// Check for valid base64url characters (A-Z, a-z, 0-9, -, _)
	re := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	if !re.MatchString(challenge) {
		return errors.New("code challenge contains invalid characters")
	}

	return nil
}

// ValidateCodeChallengeMethod validates a PKCE code challenge method
func ValidateCodeChallengeMethod(method string) bool {
	return method == "plain" || method == "S256"
}

// ValidateCodeVerifier validates a PKCE code verifier
func ValidateCodeVerifier(verifier string) error {
	if verifier == "" {
		return errors.New("code verifier cannot be empty")
	}

	// Code verifier should be 43-128 characters
	if len(verifier) < 43 || len(verifier) > 128 {
		return errors.New("code verifier must be between 43 and 128 characters")
	}

	// Check for valid characters (A-Z, a-z, 0-9, -, ., _, ~)
	re := regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
	if !re.MatchString(verifier) {
		return errors.New("code verifier contains invalid characters")
	}

	return nil
}
