package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ory/fosite"
	"github.com/ory/fosite/compose"
	"github.com/ory/fosite/storage"
)

var (
	// The OAuth 2.0 provider
	oauth2Provider fosite.OAuth2Provider

	// Private key for signing JWTs
	privateKey *rsa.PrivateKey

	// Storage implementations
	store *storage.MemoryStore

	// Session store
	sessionStore *MemorySessionStore
)

// buildOAuth2Provider creates a custom OAuth2 provider with improved client credentials support
func buildOAuth2Provider(config *fosite.Config, store fosite.Storage) fosite.OAuth2Provider {
	secret := []byte("some-super-secret-hmac-key")
	
	// Use ComposeAllEnabled for base functionality
	provider := compose.ComposeAllEnabled(config, store, secret)
	
	// ComposeAllEnabled should include:
	// - Authorization Code Grant
	// - Implicit Grant  
	// - Client Credentials Grant
	// - Refresh Token Grant
	// - OpenID Connect flows
	
	return provider
}

func main() {
	fmt.Println("Starting OAuth2 server...")
	
	// Generate RSA private key for JWT signing
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}
	fmt.Println("✓ Private key generated")

	// Initialize storage
	store = storage.NewMemoryStore()
	sessionStore = NewMemorySessionStore()
	fmt.Println("✓ Storage initialized")

	// Set up OAuth2 configuration
	config := &fosite.Config{
		AccessTokenLifespan:   time.Hour,
		RefreshTokenLifespan:  time.Hour * 24,
		AuthorizeCodeLifespan: time.Minute * 10,
		TokenURL:              "http://localhost:8080/token",
	}
	fmt.Println("✓ OAuth2 config created")

	// Initialize the OAuth2 provider with custom composition including token exchange
	oauth2Provider = buildOAuth2Provider(config, store)
	fmt.Println("✓ OAuth2 provider created with token exchange support")

	// Register clients
	registerClients()
	fmt.Println("✓ Clients registered")

	// Set up HTTP routes
	router := mux.NewRouter()

	// OAuth2 endpoints
	router.HandleFunc("/auth", authHandler).Methods("GET", "POST")
	router.HandleFunc("/token", tokenHandler).Methods("POST")
	router.HandleFunc("/callback", callbackHandler).Methods("GET")
	router.HandleFunc("/userinfo", userinfoHandler).Methods("GET", "POST")
	router.HandleFunc("/.well-known/openid_configuration", wellKnownHandler).Methods("GET")

	// Device Code Flow endpoints (RFC 8628)
	router.HandleFunc("/device_authorization", handleDeviceAuthorization).Methods("POST")
	router.HandleFunc("/device", deviceHandler).Methods("GET", "POST")

	// Sample application endpoints
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/test", serveTestPage).Methods("GET")
	router.HandleFunc("/client1/auth", client1AuthFlowHandler).Methods("GET")
	router.HandleFunc("/client2/exchange", client2TokenExchangeHandler).Methods("GET")
	router.HandleFunc("/device-flow-demo", deviceFlowDemoHandler).Methods("GET")

	// Serve static files for the device flow UI
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	fmt.Println("Starting OAuth2 server on :8080")
	fmt.Println("Authorization URL: http://localhost:8080/auth")
	fmt.Println("Token URL: http://localhost:8080/token")
	fmt.Println("Sample Client 1 (Authorization Code): http://localhost:8080/client1/auth")
	fmt.Println("Sample Client 2 (Token Exchange): http://localhost:8080/client2/exchange")
	
	log.Fatal(http.ListenAndServe(":8080", router))
}

func registerClients() {
	fmt.Println("Registering clients...")
	
	// Client 1: Authorization Code Flow Client with Device Code Flow support
	client1 := &fosite.DefaultClient{
		ID:            "frontend-client",
		Secret:        []byte("frontend-client-secret"),
		RedirectURIs:  []string{"http://localhost:8080/callback"},
		ResponseTypes: []string{"code"},
		GrantTypes:    []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
		Scopes:        []string{"openid", "profile", "email", "offline_access", "api:read"},
		Audience:      []string{"api-service", "user-service"},
	}

	// Client 2: Service Client for Client Credentials & Token Exchange
	client2 := &fosite.DefaultClient{
		ID:            "backend-client",
		Secret:        []byte("backend-client-secret"),
		RedirectURIs:  []string{},
		ResponseTypes: []string{}, // Empty for client credentials
		GrantTypes:    []string{"client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange", "refresh_token"},
		Scopes:        []string{"api:read", "api:write"},
		Audience:      []string{"api-service"},
	}

	store.Clients["frontend-client"] = client1
	store.Clients["backend-client"] = client2
	
	// Debug: Print registered clients
	fmt.Printf("Registered client1: %s with secret length: %d\n", client1.ID, len(client1.Secret))
	fmt.Printf("Registered client2: %s with secret length: %d\n", client2.ID, len(client2.Secret))
	fmt.Printf("Total clients in store: %d\n", len(store.Clients))

	// Create a sample user  
	store.Users["user1"] = storage.MemoryUserRelation{
		Username: "john.doe",
		Password: "password123",
	}
	fmt.Println("Sample user registered")
}
