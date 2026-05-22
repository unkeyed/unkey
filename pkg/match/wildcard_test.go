package match

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
)

func TestWildcard(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		pattern  string
		expected bool
		wantErr  string
		public   string
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
		{
			name:     "unicode prefix wildcard match",
			s:        "üser",
			pattern:  "ü*",
			expected: true,
		},
		{
			name:     "unicode suffix wildcard match",
			s:        "café",
			pattern:  "caf*",
			expected: true,
		},
		{
			name:     "emoji prefix wildcard match",
			s:        "🔥x",
			pattern:  "🔥*",
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
		{
			name:    "newline input returns error",
			s:       "hello\nworld",
			pattern: "hello*",
			wantErr: "wildcard input must be a single-line string",
			public:  "Wildcard matching only supports single-line values.",
		},
		{
			name:    "newline pattern returns error",
			s:       "hello world",
			pattern: "hello\nworld",
			wantErr: "wildcard pattern must be a single-line string",
			public:  "Wildcard patterns must be single-line values.",
		},
		{
			name:    "newline pattern error takes precedence",
			s:       "hello\nworld",
			pattern: "hello\nworld",
			wantErr: "wildcard pattern must be a single-line string",
			public:  "Wildcard patterns must be single-line values.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Wildcard(tt.s, tt.pattern)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				require.Equal(t, tt.public, fault.UserFacingMessage(err))
				require.False(t, result)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}
