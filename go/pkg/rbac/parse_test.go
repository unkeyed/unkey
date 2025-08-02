package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected PermissionQuery
	}{
		{
			name:     "Simple permission",
			query:    "api.key1.read_key",
			expected: S("api.key1.read_key"),
		},
		{
			name:  "Simple AND",
			query: "perm1 AND perm2",
			expected: And(
				S("perm1"),
				S("perm2"),
			),
		},
		{
			name:  "Simple OR",
			query: "perm1 OR perm2",
			expected: Or(
				S("perm1"),
				S("perm2"),
			),
		},
		{
			name:  "Complex query with precedence",
			query: "perm1 OR perm2 AND perm3",
			expected: Or(
				S("perm1"),
				And(
					S("perm2"),
					S("perm3"),
				),
			),
		},
		{
			name:  "Query with parentheses",
			query: "(perm1 OR perm2) AND perm3",
			expected: And(
				Or(
					S("perm1"),
					S("perm2"),
				),
				S("perm3"),
			),
		},
		{
			name:     "Permission with asterisk (literal)",
			query:    "api.*",
			expected: S("api.*"),
		},
		{
			name:  "Complex query with asterisk permissions",
			query: "api.* OR api.read",
			expected: Or(
				S("api.*"),
				S("api.read"),
			),
		},
		{
			name:     "Permission with multiple asterisks",
			query:    "api.*.*.read",
			expected: S("api.*.*.read"),
		},
		{
			name:     "Permission with colon namespace",
			query:    "system:admin:read",
			expected: S("system:admin:read"),
		},
		{
			name:  "Complex query with all allowed characters",
			query: "system:admin.* AND api_v2-test:*",
			expected: And(
				S("system:admin.*"),
				S("api_v2-test:*"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQuery(tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestParseQuery_Errors(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectedErr string
	}{
		{
			name:        "Empty query",
			query:       "",
			expectedErr: "query contains no tokens besides EOF",
		},
		{
			name:        "Invalid character",
			query:       "perm1 @ perm2",
			expectedErr: "invalid character '@'",
		},
		{
			name:        "Missing operand",
			query:       "perm1 AND",
			expectedErr: "reached end of input while expecting a permission or opening parenthesis",
		},
		{
			name:        "Unmatched parenthesis",
			query:       "(perm1 AND perm2",
			expectedErr: "missing closing parenthesis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseQuery(tt.query)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.expectedErr)
			require.Equal(t, PermissionQuery{}, result)
		})
	}
}
