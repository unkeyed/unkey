package main

import (
	"testing"
)

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
				t.Errorf("ToCamelInitCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
