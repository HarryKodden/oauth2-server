package config

import (
	"os"
	"testing"
)

func TestEnvironmentVariables(t *testing.T) {
	os.Setenv("TEST_ENV_VAR", "test_value")
	defer os.Unsetenv("TEST_ENV_VAR")

	value := os.Getenv("TEST_ENV_VAR")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}