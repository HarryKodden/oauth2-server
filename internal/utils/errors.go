package utils

import (
    "encoding/json"
    "net/http"
)

// WriteErrorResponse writes an OAuth2 error response
func WriteErrorResponse(w http.ResponseWriter, errorCode, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, errorCode, description)
}

// OAuth2 error responses
func WriteInvalidRequestError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, "invalid_request", description)
}

func WriteInvalidClientError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusUnauthorized, "invalid_client", description)
}

func WriteClientNotFoundError(w http.ResponseWriter, client string) {
	writeOAuth2Error(w, http.StatusNotFound, "invalid_client", client)
}

func WriteMethodNotAllowedError(w http.ResponseWriter) {
    w.Header().Set("Allow", "POST")
    writeOAuth2Error(w, http.StatusMethodNotAllowed, "invalid_request", "Method not allowed")
}

func WriteJSONError(w http.ResponseWriter) {	
	writeOAuth2Error(w, http.StatusBadRequest, "invalid_request", "Invalid JSON format")
}

func writeOAuth2Error(w http.ResponseWriter, statusCode int, errorCode, description string) {
    w.Header().Set("Content-Type", "application/json")
    w.Header().Set("Cache-Control", "no-store")
    w.Header().Set("Pragma", "no-cache")
    w.WriteHeader(statusCode)
    
    errorResponse := map[string]interface{}{
        "error":             errorCode,
        "error_description": description,
    }
    
    json.NewEncoder(w).Encode(errorResponse)
}

// Additional error types for OAuth2
func WriteUnsupportedGrantTypeError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, "unsupported_grant_type", description)
}

func WriteInvalidGrantError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, "invalid_grant", description)
}

func WriteUnauthorizedClientError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, "unauthorized_client", description)
}

func WriteInvalidScopeError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusBadRequest, "invalid_scope", description)
}

func WriteServerError(w http.ResponseWriter, description string) {
    writeOAuth2Error(w, http.StatusInternalServerError, "server_error", description)
}