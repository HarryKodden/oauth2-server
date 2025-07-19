package main

import (
	"time"
)

// Configuration holds all the configurable settings
type Configuration struct {
	// Server settings
	ServerPort string
	BaseURL    string

	// Token lifespans
	AccessTokenLifespan   time.Duration
	RefreshTokenLifespan  time.Duration
	AuthorizeCodeLifespan time.Duration
	DeviceCodeLifespan    time.Duration

	// Endpoints
	TokenURL              string
	DeviceVerificationURL string
	DeviceAuthURL         string

	// Client configurations
	FrontendClient ClientConfig
	BackendClient  ClientConfig

	// Test user credentials
	TestUser TestUserConfig
}

// ClientConfig represents OAuth2 client configuration
type ClientConfig struct {
	ID           string
	Secret       string
	GrantTypes   []string
	Scopes       []string
	Audience     []string
	RedirectURIs []string
}

// TestUserConfig represents test user configuration
type TestUserConfig struct {
	ID       string
	Username string
	Password string
	Name     string
	Email    string
}

// GetDefaultConfiguration returns the default configuration
func GetDefaultConfiguration() *Configuration {
	return &Configuration{
		// Server settings
		ServerPort: ":8080",
		BaseURL:    "http://localhost:8080",

		// Token lifespans
		AccessTokenLifespan:   time.Hour,
		RefreshTokenLifespan:  time.Hour * 24,
		AuthorizeCodeLifespan: time.Minute * 10,
		DeviceCodeLifespan:    time.Minute * 10,

		// Endpoints
		TokenURL:              "http://localhost:8080/token",
		DeviceVerificationURL: "http://localhost:8080/device",
		DeviceAuthURL:         "http://localhost:8080/device/auth",

		// Client 1: Frontend/Authorization Code Flow Client  
		FrontendClient: ClientConfig{
			ID:         "frontend-client",
			Secret:     "frontend-client-secret",
			GrantTypes: []string{"authorization_code", "refresh_token", "urn:ietf:params:oauth:grant-type:device_code"},
			Scopes:     []string{"openid", "profile", "email", "offline_access", "api:read"},
			Audience:   []string{"api-service", "user-service"},
		},

		// Client 2: Backend Client for Client Credentials & Token Exchange
		BackendClient: ClientConfig{
			ID:         "backend-client",
			Secret:     "backend-client-secret",
			GrantTypes: []string{"client_credentials", "urn:ietf:params:oauth:grant-type:token-exchange", "refresh_token"},
			Scopes:     []string{"api:read", "api:write"},
			Audience:   []string{"api-service"},
		},

		// Test user
		TestUser: TestUserConfig{
			ID:       "user1",
			Username: "john.doe",
			Password: "password123",
			Name:     "John Doe",
			Email:    "john.doe@example.com",
		},
	}
}
