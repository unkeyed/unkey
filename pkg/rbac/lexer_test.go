package rbac

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "Empty string",
			input: "",
			expected: []token{
				{typ: eof, value: "", pos: 0},
			},
		},
		{
			name:  "Single permission",
			input: "api.key1.read_key",
			expected: []token{
				{typ: permission, value: "api.key1.read_key", pos: 0},
				{typ: eof, value: "", pos: 17},
			},
		},
		{
			name:  "Simple AND",
			input: "perm1 AND perm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: eof, value: "", pos: 15},
			},
		},
		{
			name:  "Simple OR",
			input: "perm1 OR perm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: or, value: "OR", pos: 6},
				{typ: permission, value: "perm2", pos: 9},
				{typ: eof, value: "", pos: 14},
			},
		},
		{
			name:  "Parentheses",
			input: "(perm1)",
			expected: []token{
				{typ: lparen, value: "(", pos: 0},
				{typ: permission, value: "perm1", pos: 1},
				{typ: rparen, value: ")", pos: 6},
				{typ: eof, value: "", pos: 7},
			},
		},
		{
			name:  "Complex expression",
			input: "perm1 AND (perm2 OR perm3)",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 6},
				{typ: lparen, value: "(", pos: 10},
				{typ: permission, value: "perm2", pos: 11},
				{typ: or, value: "OR", pos: 17},
				{typ: permission, value: "perm3", pos: 20},
				{typ: rparen, value: ")", pos: 25},
				{typ: eof, value: "", pos: 26},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens := []token{}

			for {
				tok := lexer.nextToken()
				tokens = append(tokens, tok)
				if tok.typ == eof {
					break
				}
			}

			require.Equal(t, tt.expected, tokens)
		})
	}
}

func TestLexer_CaseInsensitiveOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "Lowercase AND",
			input: "perm1 and perm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "and", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: eof, value: "", pos: 15},
			},
		},
		{
			name:  "Lowercase OR",
			input: "perm1 or perm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: or, value: "or", pos: 6},
				{typ: permission, value: "perm2", pos: 9},
				{typ: eof, value: "", pos: 14},
			},
		},
		{
			name:  "Mixed case",
			input: "perm1 And perm2 Or perm3",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "And", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: or, value: "Or", pos: 16},
				{typ: permission, value: "perm3", pos: 19},
				{typ: eof, value: "", pos: 24},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens := []token{}

			for {
				tok := lexer.nextToken()
				tokens = append(tokens, tok)
				if tok.typ == eof {
					break
				}
			}

			require.Equal(t, tt.expected, tokens)
		})
	}
}

func TestLexer_Whitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "Extra spaces",
			input: "  perm1   AND   perm2  ",
			expected: []token{
				{typ: permission, value: "perm1", pos: 2},
				{typ: and, value: "AND", pos: 10},
				{typ: permission, value: "perm2", pos: 16},
				{typ: eof, value: "", pos: 23},
			},
		},
		{
			name:  "Tabs",
			input: "perm1\tAND\tperm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: eof, value: "", pos: 15},
			},
		},
		{
			name:  "Newlines",
			input: "perm1\nAND\nperm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: eof, value: "", pos: 15},
			},
		},
		{
			name:  "Carriage returns",
			input: "perm1\rAND\rperm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 6},
				{typ: permission, value: "perm2", pos: 10},
				{typ: eof, value: "", pos: 15},
			},
		},
		{
			name:  "Mixed whitespace",
			input: "perm1 \t\n AND \r\n perm2",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: and, value: "AND", pos: 9},
				{typ: permission, value: "perm2", pos: 16},
				{typ: eof, value: "", pos: 21},
			},
		},
		{
			name:  "Adjacent parentheses",
			input: "perm1(perm2)",
			expected: []token{
				{typ: permission, value: "perm1", pos: 0},
				{typ: lparen, value: "(", pos: 5},
				{typ: permission, value: "perm2", pos: 6},
				{typ: rparen, value: ")", pos: 11},
				{typ: eof, value: "", pos: 12},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens := []token{}

			for {
				tok := lexer.nextToken()
				tokens = append(tokens, tok)
				if tok.typ == eof {
					break
				}
			}

			require.Equal(t, tt.expected, tokens)
		})
	}
}

func TestLexer_PermissionFormats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "Permission with dots",
			input: "api.key1.read_key",
			expected: []token{
				{typ: permission, value: "api.key1.read_key", pos: 0},
				{typ: eof, value: "", pos: 17},
			},
		},
		{
			name:  "Permission with underscores",
			input: "namespace.ns_1.create_override",
			expected: []token{
				{typ: permission, value: "namespace.ns_1.create_override", pos: 0},
				{typ: eof, value: "", pos: 30},
			},
		},
		{
			name:  "Permission with hyphens",
			input: "api.key-123.read_key",
			expected: []token{
				{typ: permission, value: "api.key-123.read_key", pos: 0},
				{typ: eof, value: "", pos: 20},
			},
		},
		{
			name:  "Permission with numbers",
			input: "api.key123.read_key",
			expected: []token{
				{typ: permission, value: "api.key123.read_key", pos: 0},
				{typ: eof, value: "", pos: 19},
			},
		},
		{
			name:  "Mixed valid characters",
			input: "api123.key_456-test.read_key",
			expected: []token{
				{typ: permission, value: "api123.key_456-test.read_key", pos: 0},
				{typ: eof, value: "", pos: 28},
			},
		},
		{
			name:  "Permission with slash",
			input: "api/v1.read_key",
			expected: []token{
				{typ: permission, value: "api/v1.read_key", pos: 0},
				{typ: eof, value: "", pos: 15},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens := []token{}

			for {
				tok := lexer.nextToken()
				tokens = append(tokens, tok)
				if tok.typ == eof {
					break
				}
			}

			require.Equal(t, tt.expected, tokens)
		})
	}
}

func TestLexer_Errors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "Invalid character",
			input:       "perm1 @ perm2",
			expectedErr: "invalid character '@' at position 6",
		},
		{
			name:        "Permission with special char",
			input:       "api.key1.read@key",
			expectedErr: "invalid character '@' at position 13",
		},
		{
			name:        "Invalid character dollar sign",
			input:       "api.key1$ AND perm2",
			expectedErr: "invalid character '$' at position 8",
		},
		{
			name:        "Invalid character hash",
			input:       "perm1 # perm2",
			expectedErr: "invalid character '#' at position 6",
		},
		{
			name:        "Invalid character percent",
			input:       "perm1 % perm2",
			expectedErr: "invalid character '%' at position 6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)

			var foundError bool
			for {
				tok := lexer.nextToken()
				if tok.typ == errorToken {
					foundError = true
					require.Contains(t, tok.value, tt.expectedErr)
					break
				}
				if tok.typ == eof {
					break
				}
			}

			require.True(t, foundError, "Expected an error token but got none")
		})
	}
}

func TestLexer_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []token
	}{
		{
			name:  "Only whitespace",
			input: "   \t\n  ",
			expected: []token{
				{typ: eof, value: "", pos: 7},
			},
		},
		{
			name:  "Single character permission",
			input: "a",
			expected: []token{
				{typ: permission, value: "a", pos: 0},
				{typ: eof, value: "", pos: 1},
			},
		},
		{
			name:  "Operators as permissions (should not be recognized)",
			input: "ANDperm ORperm",
			expected: []token{
				{typ: permission, value: "ANDperm", pos: 0},
				{typ: permission, value: "ORperm", pos: 8},
				{typ: eof, value: "", pos: 14},
			},
		},
		{
			name:  "Nested parentheses",
			input: "((perm1))",
			expected: []token{
				{typ: lparen, value: "(", pos: 0},
				{typ: lparen, value: "(", pos: 1},
				{typ: permission, value: "perm1", pos: 2},
				{typ: rparen, value: ")", pos: 7},
				{typ: rparen, value: ")", pos: 8},
				{typ: eof, value: "", pos: 9},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lexer := newLexer(tt.input)
			tokens := []token{}

			for {
				tok := lexer.nextToken()
				tokens = append(tokens, tok)
				if tok.typ == eof || tok.typ == errorToken {
					break
				}
			}

			require.Equal(t, tt.expected, tokens)
		})
	}
}
