package config

import (
	"fmt"
	"oauth2-server/internal/utils"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

func (c *Config) NormalizeAllClientRedirectURIs() {
	for i := range c.Clients {
		for j, uri := range c.Clients[i].RedirectURIs {
			c.Clients[i].RedirectURIs[j] = utils.NormalizeRedirectURI(c.Server.BaseURL, uri)
		}
	}
}

// LoadConfig loads configuration from environment variables and config file
func Load() (*Config, error) {
	cfg := &Config{}

	// 1. Load YAML config
	configPath := getEnv("CONFIG_FILE", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		if err := LoadFromFile(configPath, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// 2. Apply environment variable overrides
	cfg.LoadFromEnv()

	// 3. Normalize redirect URIs with the final base URL
	cfg.NormalizeAllClientRedirectURIs()

	cfg.BaseURL = cfg.Server.BaseURL
	cfg.Port = fmt.Sprintf("%d", cfg.Server.Port)
	cfg.Host = cfg.Server.Host

	return cfg, nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
