package utils

import (
	"net/http"
	"strings"
)

func ValidateClientRedirectURI(requestedURI string, registeredURIs []string) bool {
	for _, registeredURI := range registeredURIs {
		if requestedURI == registeredURI {
			return true
		}
	}
	return false
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

// NormalizeRedirectURI converts a relative URI to an absolute URI using the base URL.
// If the URI is already absolute, it is returned unchanged.
func NormalizeRedirectURI(baseURL, uri string) string {
	if uri == "" {
		return ""
	}
	if strings.HasPrefix(uri, "/") {
		return strings.TrimRight(baseURL, "/") + uri
	}
	if !strings.Contains(uri, "://") {
		return baseURL + "/" + strings.TrimLeft(uri, "/")
	}
	return uri
}
