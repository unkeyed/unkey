package main

import (
	"testing"
)

func TestPluralize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Regular plurals
		{"cat", "cats"},
		{"dog", "dogs"},
		{"Key", "Keys"},
		{"User", "Users"},
		{"AuditLog", "AuditLogs"},

		// Words ending in s, sh, ch, x, z
		{"class", "classes"},
		{"dish", "dishes"},
		{"church", "churches"},
		{"box", "boxes"},
		{"buzz", "buzzes"},

		// Words ending in y
		{"city", "cities"},
		{"baby", "babies"},
		{"boy", "boys"}, // vowel before y
		{"key", "keys"}, // vowel before y

		// Edge cases
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := pluralize(tt.input)
			if result != tt.expected {
				t.Errorf("pluralize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{

		{"user_id", "UserID"},
		{"name", "Name"},
		{"created_at", "CreatedAt"},

		// Special abbreviations
		{"id", "ID"},

		// Compound names with URL (should be Url at the end)
		{"git_repository_url", "GitRepositoryUrl"},
		{"profile_image_url", "ProfileImageUrl"},

		// Multiple underscores
		{"some_long_field_name", "SomeLongFieldName"},

		// Edge cases
		{"", ""},
		{"a", "A"},
		{"a_b", "AB"},

		// Mixed with numbers
		{"user_id_2", "UserID2"},
		{"api_key", "ApiKey"},
		{"http_status", "HttpStatus"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
