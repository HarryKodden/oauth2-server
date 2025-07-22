package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"oauth2-server/internal/store"
	"oauth2-server/pkg/config"
)

// DebugHandlers provides debugging endpoints
type DebugHandlers struct {
	clientStore *store.ClientStore
	config      *config.Config
}

// NewDebugHandlers creates a new debug handlers instance
func NewDebugHandlers(clientStore *store.ClientStore, cfg *config.Config) *DebugHandlers {
	return &DebugHandlers{
		clientStore: clientStore,
		config:      cfg,
	}
}

// HandleDebugClients lists all configured clients (for debugging)
func (h *DebugHandlers) HandleDebugClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("🔍 Debug: Listing all configured clients")

	// Get all clients from config
	var clientList []map[string]interface{}

	for _, clientConfig := range h.config.Clients {
		clientInfo := map[string]interface{}{
			"id":                         clientConfig.ID,
			"name":                       clientConfig.Name,
			"description":                clientConfig.Description,
			"redirect_uris":              clientConfig.RedirectURIs,
			"grant_types":                clientConfig.GrantTypes,
			"response_types":             clientConfig.ResponseTypes,
			"scopes":                     clientConfig.Scopes,
			"audience":                   clientConfig.Audience,
			"token_endpoint_auth_method": clientConfig.TokenEndpointAuthMethod,
			"public":                     clientConfig.Public,
			"enabled_flows":              clientConfig.EnabledFlows,
			"has_secret":                 clientConfig.Secret != "",
		}
		clientList = append(clientList, clientInfo)
	}

	response := map[string]interface{}{
		"total_clients": len(clientList),
		"clients":       clientList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Debug: Listed %d clients", len(clientList))
}

// HandleDebugClient shows details for a specific client
func (h *DebugHandlers) HandleDebugClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id parameter is required", http.StatusBadRequest)
		return
	}

	log.Printf("🔍 Debug: Looking up client: %s", clientID)

	// Check if client exists in config
	var foundClient *config.ClientConfig
	for _, client := range h.config.Clients {
		if client.ID == clientID {
			foundClient = &client
			break
		}
	}

	if foundClient == nil {
		response := map[string]interface{}{
			"error":       "client_not_found",
			"message":     "Client not found in configuration",
			"client_id":   clientID,
			"searched_in": "config.yaml",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if client exists in store
	storeClient, err := h.clientStore.GetClient(r.Context(), clientID)
	var storeExists bool
	var storeError string
	if err != nil {
		storeError = err.Error()
		storeExists = false
	} else {
		storeExists = storeClient != nil
	}

	response := map[string]interface{}{
		"client_id":                  foundClient.ID,
		"name":                       foundClient.Name,
		"description":                foundClient.Description,
		"redirect_uris":              foundClient.RedirectURIs,
		"grant_types":                foundClient.GrantTypes,
		"response_types":             foundClient.ResponseTypes,
		"scopes":                     foundClient.Scopes,
		"audience":                   foundClient.Audience,
		"token_endpoint_auth_method": foundClient.TokenEndpointAuthMethod,
		"public":                     foundClient.Public,
		"enabled_flows":              foundClient.EnabledFlows,
		"has_secret":                 foundClient.Secret != "",
		"config_status": map[string]interface{}{
			"found_in_config": true,
			"found_in_store":  storeExists,
			"store_error":     storeError,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Debug: Client %s found in config, store_exists=%t", clientID, storeExists)
}

// HandleDebugConfig shows current configuration
func (h *DebugHandlers) HandleDebugConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("🔍 Debug: Showing current configuration")

	response := map[string]interface{}{
		"server": map[string]interface{}{
			"base_url":         h.config.Server.BaseURL,
			"port":             h.config.Server.Port,
			"host":             h.config.Server.Host,
			"read_timeout":     h.config.Server.ReadTimeout,
			"write_timeout":    h.config.Server.WriteTimeout,
			"shutdown_timeout": h.config.Server.ShutdownTimeout,
		},
		"security": map[string]interface{}{
			"token_expiry_seconds":         h.config.Security.TokenExpirySeconds,
			"refresh_token_expiry_seconds": h.config.Security.RefreshTokenExpirySeconds,
			"device_code_expiry_seconds":   h.config.Security.DeviceCodeExpirySeconds,
			"enable_pkce":                  h.config.Security.EnablePKCE,
			"require_https":                h.config.Security.RequireHTTPS,
			"has_jwt_secret":               h.config.Security.JWTSecret != "",
		},
		"clients_count": len(h.config.Clients),
		"users_count":   len(h.config.Users),
		"logging": map[string]interface{}{
			"level":        h.config.Logging.Level,
			"format":       h.config.Logging.Format,
			"enable_audit": h.config.Logging.EnableAudit,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	log.Printf("✅ Debug: Configuration displayed")
}
