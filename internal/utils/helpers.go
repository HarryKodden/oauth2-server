package utils

import (
	"errors"
	"net/http"
	"strings"
)

// ExtractBearerToken extracts the token from Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
    parts := strings.Split(authHeader, " ")
    if len(parts) != 2 || parts[0] != "Bearer" {
        return "", errors.New("invalid authorization header format")
    }
    return parts[1], nil
}

// ExtractClientCredentials extracts client credentials from request
func ExtractClientCredentials(r *http.Request) (string, string, error) {
    // Try basic auth first
    if clientID, clientSecret, ok := r.BasicAuth(); ok {
        return clientID, clientSecret, nil
    }
    
    // Try form parameters
    clientID := r.FormValue("client_id")
    clientSecret := r.FormValue("client_secret")
    
    if clientID == "" {
        return "", "", errors.New("client_id is required")
    }
    
    return clientID, clientSecret, nil
}


// Contains checks if a slice contains a specific string
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// RemoveDuplicates removes duplicate strings from a slice
func RemoveDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// IsValidRedirectURI validates a redirect URI
func IsValidRedirectURI(uri string) bool {
	if uri == "" {
		return false
	}

	// Basic URL validation - in a real implementation, you'd be more thorough
	return strings.HasPrefix(uri, "http://") ||
		strings.HasPrefix(uri, "https://") ||
		strings.Contains(uri, "://") // Allow custom schemes for mobile apps
}

// SplitScopes splits a space-separated scope string into individual scopes
func SplitScopes(scopes string) []string {
	if scopes == "" {
		return []string{}
	}
	return strings.Fields(scopes)
}

// JoinScopes joins individual scopes into a space-separated string
func JoinScopes(scopes []string) string {
	return strings.Join(scopes, " ")
}

// NormalizeScope normalizes a scope string by removing duplicates and sorting
func NormalizeScope(scopes string) string {
	scopeList := SplitScopes(scopes)
	scopeList = RemoveDuplicates(scopeList)
	return JoinScopes(scopeList)
}

// FilterScopes filters requested scopes against allowed scopes
func FilterScopes(requestedScopes, allowedScopes []string) []string {
	var filtered []string

	for _, requested := range requestedScopes {
		for _, allowed := range allowedScopes {
			if requested == allowed {
				filtered = append(filtered, requested)
				break
			}
		}
	}

	return filtered
}

// ExtractClientIDFromPath extracts client ID from URL path
func ExtractClientIDFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "clients" {
		return parts[1]
	}
	return ""
}

// ValidateRegistrationAccessToken checks if the registration access token is valid
func ValidateRegistrationAccessToken(token string) bool {
	// Simplified validation - in a real implementation, you'd validate JWT
	return token != "" && len(token) > 10
}

// GetRequestBaseURL determines the base URL from the HTTP request, considering proxy headers
func GetRequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Check for proxy headers
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}

	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}

	return scheme + "://" + host
}

// GetEffectiveBaseURL returns the effective base URL considering configuration and proxy headers
func GetEffectiveBaseURL(configBaseURL string, r *http.Request) string {
	if configBaseURL != "" {
		return configBaseURL
	}
	return GetRequestBaseURL(r)
}

