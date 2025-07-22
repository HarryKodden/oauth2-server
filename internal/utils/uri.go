package utils

import (
	"net/http"
	"net/url"
	"strings"
)

// ResolveRedirectURIs resolves relative redirect URIs to absolute URIs
func ResolveRedirectURIs(redirectURIs []string, r *http.Request, configBaseURL string) []string {
	if len(redirectURIs) == 0 {
		return []string{}
	}

	// Get the effective base URL (proxy-aware)
	baseURL := GetRequestBaseURL(r)
	if configBaseURL != "" {
		baseURL = configBaseURL // Override if explicitly provided
	}

	resolved := make([]string, 0, len(redirectURIs))

	for _, uri := range redirectURIs {
		if uri == "" {
			continue
		}

		// If URI is already absolute, keep it as-is
		if strings.Contains(uri, "://") {
			resolved = append(resolved, uri)
			continue
		}

		// If URI starts with /, it's relative to the root
		if strings.HasPrefix(uri, "/") {
			resolved = append(resolved, baseURL+uri)
		} else {
			// If URI doesn't start with /, treat it as relative to root
			resolved = append(resolved, baseURL+"/"+uri)
		}
	}

	return resolved
}

// ValidateClientRedirectURI validates a redirect URI against registered URIs
func ValidateClientRedirectURI(requestedURI string, registeredURIs []string, r *http.Request, configBaseURL string) bool {
	// Resolve registered URIs to absolute URIs
	resolvedURIs := ResolveRedirectURIs(registeredURIs, r, configBaseURL)

	// Check exact match with resolved URIs
	for _, resolvedURI := range resolvedURIs {
		if requestedURI == resolvedURI {
			return true
		}
	}

	// Also check against original URIs (for absolute URIs that don't need resolution)
	for _, registeredURI := range registeredURIs {
		if requestedURI == registeredURI {
			return true
		}
	}

	return false
}

// NormalizeRedirectURI normalizes a redirect URI for comparison
func NormalizeRedirectURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	// Remove default ports
	if (parsed.Scheme == "http" && parsed.Port() == "80") ||
		(parsed.Scheme == "https" && parsed.Port() == "443") {
		parsed.Host = parsed.Hostname()
	}

	return parsed.String()
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

	port := r.Header.Get("X-Forwarded-Port")

	// Build the base URL
	if port != "" && port != "80" && port != "443" {
		return scheme + "://" + host + ":" + port
	}

	return scheme + "://" + host
}
