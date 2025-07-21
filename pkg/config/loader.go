package config

import (
    "fmt"
    "io/ioutil"
    "os"
    "strconv"

    "gopkg.in/yaml.v2"
)

// LoadConfig loads configuration from environment variables and config file
func Load() (*Config, error) {
    cfg := &Config{}
    
    // Set defaults
    cfg.Server.Port = getEnvAsInt("SERVER_PORT", 8080)
    cfg.Server.Host = getEnv("SERVER_HOST", "localhost")
    cfg.Server.BaseURL = getEnv("SERVER_BASE_URL", fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port))
    cfg.Server.ReadTimeout = getEnvAsInt("SERVER_READ_TIMEOUT", 30)
    cfg.Server.WriteTimeout = getEnvAsInt("SERVER_WRITE_TIMEOUT", 30)
    cfg.Server.ShutdownTimeout = getEnvAsInt("SERVER_SHUTDOWN_TIMEOUT", 5)
    
    cfg.Security.JWTSecret = getEnv("JWT_SECRET", "your-secret-key-here")
    cfg.Security.TokenExpirySeconds = getEnvAsInt("TOKEN_EXPIRY_SECONDS", 3600)
    cfg.Security.RefreshTokenExpirySeconds = getEnvAsInt("REFRESH_TOKEN_EXPIRY_SECONDS", 86400)
    cfg.Security.DeviceCodeExpirySeconds = getEnvAsInt("DEVICE_CODE_EXPIRY_SECONDS", 600)
    cfg.Security.EnablePKCE = getEnvAsBool("ENABLE_PKCE", true)
    cfg.Security.RequireHTTPS = getEnvAsBool("REQUIRE_HTTPS", false)
    
    cfg.Logging.Level = getEnv("LOG_LEVEL", "info")
    cfg.Logging.Format = getEnv("LOG_FORMAT", "text")
    cfg.Logging.EnableAudit = getEnvAsBool("ENABLE_AUDIT", true)
    
    // Try to load from config file
    configPath := getEnv("CONFIG_FILE", "config.yaml")
    if _, err := os.Stat(configPath); err == nil {
        if err := LoadFromFile(configPath, cfg); err != nil {
            return nil, fmt.Errorf("failed to load config file: %w", err)
        }
    }
    
    return cfg, nil
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string, cfg *Config) error {
    data, err := ioutil.ReadFile(path)
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