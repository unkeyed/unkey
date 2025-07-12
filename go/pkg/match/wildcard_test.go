package match

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWildcard(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pattern  string
		expected bool
	}{
		// Email patterns
		{
			name:     "email wildcard match gmail",
			s:        "test@gmail.com",
			pattern:  "*@gmail.com",
			expected: true,
		},
		{
			name:     "email wildcard no match different domain",
			s:        "test@yahoo.com",
			pattern:  "*@gmail.com",
			expected: false,
		},
		{
			name:     "email exact match",
			s:        "test@gmail.com",
			pattern:  "test@gmail.com",
			expected: true,
		},
		// Prefix patterns
		{
			name:     "prefix wildcard match",
			s:        "hello world",
			pattern:  "hello*",
			expected: true,
		},
		{
			name:     "prefix wildcard no match",
			s:        "goodbye world",
			pattern:  "hello*",
			expected: false,
		},
		// Suffix patterns
		{
			name:     "suffix wildcard match",
			s:        "hello world",
			pattern:  "*world",
			expected: true,
		},
		{
			name:     "suffix wildcard no match",
			s:        "hello earth",
			pattern:  "*world",
			expected: false,
		},
		// Middle patterns
		{
			name:     "middle wildcard match",
			s:        "hello world",
			pattern:  "h*d",
			expected: true,
		},
		{
			name:     "multiple wildcards",
			s:        "hello beautiful world",
			pattern:  "h*beau*world",
			expected: true,
		},
		// Special regex characters
		{
			name:     "dots are literal",
			s:        "test@gmail.com",
			pattern:  "*@gmail.com",
			expected: true,
		},
		{
			name:     "dots must match exactly",
			s:        "test@gmailxcom",
			pattern:  "*@gmail.com",
			expected: false,
		},
		// Edge cases
		{
			name:     "empty string with wildcard",
			s:        "",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "wildcard only",
			s:        "anything",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "no wildcard exact match",
			s:        "exact",
			pattern:  "exact",
			expected: true,
		},
		{
			name:     "no wildcard no match",
			s:        "different",
			pattern:  "exact",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Wildcard(tt.s, tt.pattern)
			require.NoError(t, err)
			if result != tt.expected {
				t.Errorf("Wildcard(%q, %q) = %v, want %v", tt.s, tt.pattern, result, tt.expected)
			}
		})
	}
}
