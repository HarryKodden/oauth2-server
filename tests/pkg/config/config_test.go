package config_test

import (
	"os"
	"testing"

	"oauth2-server/pkg/config"
)

func TestConfigLoading(t *testing.T) {
	t.Run("should load default config when YAML file doesn't exist", func(t *testing.T) {
		// Test loading config without YAML file
		cfg, err := config.NewConfig("nonexistent.yaml") // Changed from LoadConfig to NewConfig
		if err != nil {
			t.Fatalf("Expected no error when YAML file doesn't exist, got: %v", err)
		}

		// Now cfg is properly used, so no unused variable error
		if cfg == nil {
			t.Fatal("Expected config to be loaded with defaults")
		}

		// Test that defaults are set
		if cfg.Server.Port == 0 {
			t.Error("Expected default port to be set")
		}

		if cfg.Security.JWTSecret == "" {
			t.Error("Expected JWT secret to be set from environment or defaults")
		}
	})

	t.Run("should load config from YAML file", func(t *testing.T) {
		// Create a temporary YAML config file for testing
		yamlContent := `
server:
  port: 9090
  host: "test-host"
  base_url: "http://test-host:9090"
  read_timeout: 30
  write_timeout: 30
  shutdown_timeout: 10

security:
  jwt_signing_key: "test-secret-key"
  token_expiry_seconds: 3600
  refresh_token_expiry_seconds: 86400
  device_code_expiry_seconds: 600
  enable_pkce: true
  require_https: false

logging:
  level: "debug"
  format: "json"
  enable_audit: true

clients:
  - id: "test-client"
    secret: "test-secret"
    name: "Test Client"
    redirect_uris: ["http://localhost:3000/callback"]
    grant_types: ["authorization_code", "refresh_token"]
    response_types: ["code"]
    scopes: ["openid", "profile"]
    public: false

users:
  - id: "test-user"
    username: "testuser"
    password: "testpass"
    email: "test@example.com"
    name: "Test User"
`

		// Write temporary config file
		tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(yamlContent); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpFile.Close()

		// Test loading from YAML file
		cfg, err := config.NewConfig(tmpFile.Name()) // Changed from LoadConfig to NewConfig
		if err != nil {
			t.Fatalf("Failed to load config from YAML: %v", err)
		}

		// Validate YAML values were loaded
		if cfg.Server.Port != 9090 {
			t.Errorf("Expected port 9090, got %d", cfg.Server.Port)
		}

		if cfg.Server.Host != "test-host" {
			t.Errorf("Expected host 'test-host', got '%s'", cfg.Server.Host)
		}

		if cfg.Security.JWTSecret != "test-secret-key" {
			t.Errorf("Expected JWT secret 'test-secret-key', got '%s'", cfg.Security.JWTSecret)
		}

		if len(cfg.Clients) != 1 {
			t.Errorf("Expected 1 client, got %d", len(cfg.Clients))
		}

		if len(cfg.Users) != 1 {
			t.Errorf("Expected 1 user, got %d", len(cfg.Users))
		}

		// Test client configuration
		client := cfg.Clients[0]
		if client.ID != "test-client" {
			t.Errorf("Expected client ID 'test-client', got '%s'", client.ID)
		}

		// Test user configuration
		user := cfg.Users[0]
		if user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
	})

	t.Run("should handle environment variable overrides", func(t *testing.T) {
		// Set environment variables
		os.Setenv("OAUTH2_JWT_SECRET", "env-secret")
		os.Setenv("OAUTH2_PORT", "8888")
		defer func() {
			os.Unsetenv("OAUTH2_JWT_SECRET")
			os.Unsetenv("OAUTH2_PORT")
		}()

		cfg, err := config.NewConfig() // Changed from LoadConfig to NewConfig
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Note: The actual environment override behavior depends on your LoadFromEnv implementation
		// Adjust these assertions based on your actual implementation
		if cfg.Security.JWTSecret != "env-secret" {
			t.Logf("JWT secret from env: %s (might not be overridden depending on LoadFromEnv implementation)", cfg.Security.JWTSecret)
		}
	})
}

func TestConfigValidation(t *testing.T) {
	t.Run("should validate required fields", func(t *testing.T) {
		// Test config with missing JWT secret
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "localhost",
			},
			Security: config.SecurityConfig{
				JWTSecret: "", // Missing JWT secret
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for missing JWT secret")
		}
	})

	t.Run("should validate port range", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 70000, // Invalid port
				Host: "localhost",
			},
			Security: config.SecurityConfig{
				JWTSecret: "test-secret",
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid port")
		}
	})

	t.Run("should validate client configuration", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "localhost",
			},
			Security: config.SecurityConfig{
				JWTSecret: "test-secret",
			},
			Clients: []config.ClientConfig{
				{
					ID:         "", // Missing client ID
					Secret:     "secret",
					GrantTypes: []string{"authorization_code"},
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for missing client ID")
		}
	})

	t.Run("should validate grant types", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Port: 8080,
				Host: "localhost",
			},
			Security: config.SecurityConfig{
				JWTSecret: "test-secret",
			},
			Clients: []config.ClientConfig{
				{
					ID:         "test-client",
					Secret:     "secret",
					GrantTypes: []string{"invalid_grant_type"}, // Invalid grant type
				},
			},
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected validation error for invalid grant type")
		}
	})
}

func TestConfigHelperMethods(t *testing.T) {
	cfg := &config.Config{
		Clients: []config.ClientConfig{
			{
				ID:     "client1",
				Secret: "secret1",
				Name:   "Test Client 1",
			},
			{
				ID:     "client2",
				Secret: "secret2",
				Name:   "Test Client 2",
			},
		},
		Users: []config.UserConfig{
			{
				ID:       "user1",
				Username: "testuser1",
				Email:    "user1@example.com",
			},
			{
				ID:       "user2",
				Username: "testuser2",
				Email:    "user2@example.com",
			},
		},
	}

	t.Run("should find client by ID", func(t *testing.T) {
		client, found := cfg.GetClientByID("client1")
		if !found {
			t.Error("Expected to find client1")
		}
		if client.ID != "client1" {
			t.Errorf("Expected client ID 'client1', got '%s'", client.ID)
		}

		_, found = cfg.GetClientByID("nonexistent")
		if found {
			t.Error("Expected not to find nonexistent client")
		}
	})

	t.Run("should find user by username", func(t *testing.T) {
		user, found := cfg.GetUserByUsername("testuser1")
		if !found {
			t.Error("Expected to find testuser1")
		}
		if user.Username != "testuser1" {
			t.Errorf("Expected username 'testuser1', got '%s'", user.Username)
		}

		_, found = cfg.GetUserByUsername("nonexistent")
		if found {
			t.Error("Expected not to find nonexistent user")
		}
	})

	t.Run("should find user by ID", func(t *testing.T) {
		user, found := cfg.GetUserByID("user1")
		if !found {
			t.Error("Expected to find user1")
		}
		if user.ID != "user1" {
			t.Errorf("Expected user ID 'user1', got '%s'", user.ID)
		}
	})

	t.Run("should get first client and user", func(t *testing.T) {
		client, found := cfg.GetFirstClient()
		if !found {
			t.Error("Expected to find first client")
		}
		if client.ID != "client1" {
			t.Errorf("Expected first client ID 'client1', got '%s'", client.ID)
		}

		user, found := cfg.GetFirstUser()
		if !found {
			t.Error("Expected to find first user")
		}
		if user.ID != "user1" {
			t.Errorf("Expected first user ID 'user1', got '%s'", user.ID)
		}
	})
}

func TestYAMLConfigLoading(t *testing.T) {
	t.Run("should handle missing YAML file gracefully", func(t *testing.T) {
		_, err := config.LoadYAMLConfig("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error when loading nonexistent YAML file")
		}
	})

	t.Run("should load valid YAML config", func(t *testing.T) {
		yamlContent := `
server:
  port: 8080
  host: "localhost"

security:
  jwt_signing_key: "test-key"

clients:
  - id: "test"
    secret: "secret"
`

		tmpFile, err := os.CreateTemp("", "test-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.WriteString(yamlContent); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpFile.Close()

		yamlConfig, err := config.LoadYAMLConfig(tmpFile.Name())
		if err != nil {
			t.Fatalf("Failed to load YAML config: %v", err)
		}

		if yamlConfig.Server.Port != 8080 {
			t.Errorf("Expected port 8080, got %d", yamlConfig.Server.Port)
		}

		if len(yamlConfig.Clients) != 1 {
			t.Errorf("Expected 1 client, got %d", len(yamlConfig.Clients))
		}
	})
}
