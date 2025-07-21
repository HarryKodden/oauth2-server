package utils_test

import (
	"testing"

	"oauth2-server/internal/utils"
)

func TestSplitScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single_scope",
			input:    "openid",
			expected: []string{"openid"},
		},
		{
			name:     "multiple_scopes",
			input:    "openid profile email",
			expected: []string{"openid", "profile", "email"},
		},
		{
			name:     "empty_scope",
			input:    "",
			expected: []string{},
		},
		{
			name:     "scope_with_extra_spaces",
			input:    "  openid   profile   email  ",
			expected: []string{"openid", "profile", "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SplitScopes(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d scopes, got %d", len(tt.expected), len(result))
				return
			}
			for i, scope := range result {
				if scope != tt.expected[i] {
					t.Errorf("Expected scope %s at index %d, got %s", tt.expected[i], i, scope)
				}
			}
		})
	}
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "single_scope",
			input:    []string{"openid"},
			expected: "openid",
		},
		{
			name:     "multiple_scopes",
			input:    []string{"openid", "profile", "email"},
			expected: "openid profile email",
		},
		{
			name:     "empty_scopes",
			input:    []string{},
			expected: "",
		},
		{
			name:     "scopes_with_empty_strings",
			input:    []string{"openid", "", "profile", "email"},
			expected: "openid profile email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.JoinScopes(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFilterScopes(t *testing.T) {
	tests := []struct {
		name        string
		requested   []string
		allowed     []string
		expected    []string
	}{
		{
			name:      "all_scopes_allowed",
			requested: []string{"openid", "profile"},
			allowed:   []string{"openid", "profile", "email"},
			expected:  []string{"openid", "profile"},
		},
		{
			name:      "some_scopes_filtered",
			requested: []string{"openid", "profile", "admin"},
			allowed:   []string{"openid", "profile"},
			expected:  []string{"openid", "profile"},
		},
		{
			name:      "no_scopes_allowed",
			requested: []string{"admin", "superuser"},
			allowed:   []string{"openid", "profile"},
			expected:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FilterScopes(tt.requested, tt.allowed)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d scopes, got %d", len(tt.expected), len(result))
				return
			}
			for i, scope := range result {
				if scope != tt.expected[i] {
					t.Errorf("Expected scope %s at index %d, got %s", tt.expected[i], i, scope)
				}
			}
		})
	}
}