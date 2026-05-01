package middleware

import (
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// NormalizeEscapedQuerySeparators converts HTML-escaped query separators to
// plain '&' so net/url parsing does not fail on malformed incoming URLs.
// It only mutates URL query data and returns true when a change was applied.
func NormalizeEscapedQuerySeparators(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	rawQuery := r.URL.RawQuery
	if rawQuery == "" {
		return false
	}

	normalized := strings.ReplaceAll(rawQuery, "&amp;", "&")
	normalized = strings.ReplaceAll(normalized, `\u0026amp;`, "&")

	if normalized == rawQuery {
		return false
	}

	r.URL.RawQuery = normalized
	return true
}

// APIKeyAuth middleware for API key authentication
func APIKeyAuth(log *logrus.Logger, apiKey string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Allow OPTIONS requests through for CORS preflight
			if r.Method == "OPTIONS" {
				log.Printf("🔄 Allowing OPTIONS request through for CORS preflight")
				next.ServeHTTP(w, r)
				return
			}

			if apiKey == "" {
				log.Printf("⚠️  API key authentication disabled (no API key configured: '%s')", apiKey)
				next.ServeHTTP(w, r)
				return
			}

			// Check for API key in header
			authHeader := strings.TrimSpace(r.Header.Get("X-API-Key"))
			if authHeader == "" {
				log.Printf("❌ API key authentication failed: missing X-API-Key header")
				http.Error(w, "API key required", http.StatusUnauthorized)
				return
			}

			if authHeader != strings.TrimSpace(apiKey) {
				log.Printf("❌ API key authentication failed: invalid API key")
				http.Error(w, "Invalid API key", http.StatusUnauthorized)
				return
			}

			log.Printf("✅ API key authentication successful")
			next.ServeHTTP(w, r)
		}
	}
}
