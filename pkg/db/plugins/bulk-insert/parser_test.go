package main

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestSQLParser_Parse(t *testing.T) {
	parser := NewSQLParser()

	tests := []struct {
		name     string
		query    *plugin.Query
		expected *ParsedQuery
	}{
		{
			name: "simple insert",
			query: &plugin.Query{
				Text: "INSERT INTO users (id, name) VALUES (?, ?)",
			},
			expected: &ParsedQuery{
				InsertPart:           "INSERT INTO users (id, name)",
				ValuesPart:           "(?, ?)",
				OnDuplicateKeyUpdate: "",
			},
		},
		{
			name: "insert with on duplicate key update",
			query: &plugin.Query{
				Text: "INSERT INTO users (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name = VALUES(name)",
			},
			expected: &ParsedQuery{
				InsertPart:           "INSERT INTO users (id, name)",
				ValuesPart:           "(?, ?)",
				OnDuplicateKeyUpdate: "ON DUPLICATE KEY UPDATE name = VALUES(name)",
			},
		},
		{
			name: "multiline insert with comments",
			query: &plugin.Query{
				Text: `-- Insert a new user
INSERT INTO users (
    id,
    name,
    created_at -- timestamp
) VALUES (
    ?, ?, ?
)`,
			},
			expected: &ParsedQuery{
				InsertPart:           "INSERT INTO users ( id, name, created_at )",
				ValuesPart:           "( ?, ?, ? )",
				OnDuplicateKeyUpdate: "",
			},
		},
		{
			name: "insert with backticks",
			query: &plugin.Query{
				Text: "INSERT INTO `users` (`id`, `name`) VALUES (?, ?)",
			},
			expected: &ParsedQuery{
				InsertPart:           "INSERT INTO `users` (`id`, `name`)",
				ValuesPart:           "(?, ?)",
				OnDuplicateKeyUpdate: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.Parse(tt.query)

			if result.InsertPart != tt.expected.InsertPart {
				t.Errorf("InsertPart = %q, want %q", result.InsertPart, tt.expected.InsertPart)
			}
			if result.ValuesPart != tt.expected.ValuesPart {
				t.Errorf("ValuesPart = %q, want %q", result.ValuesPart, tt.expected.ValuesPart)
			}
			if result.OnDuplicateKeyUpdate != tt.expected.OnDuplicateKeyUpdate {
				t.Errorf("OnDuplicateKeyUpdate = %q, want %q", result.OnDuplicateKeyUpdate, tt.expected.OnDuplicateKeyUpdate)
			}
		})
	}
}

func TestSQLParser_cleanSQL(t *testing.T) {
	parser := NewSQLParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple query",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
		{
			name: "query with comments",
			input: `-- This is a comment
SELECT * FROM users -- Another comment
WHERE id = 1`,
			expected: "SELECT * FROM users WHERE id = 1",
		},
		{
			name: "multiline query",
			input: `SELECT *
FROM users
WHERE id = 1`,
			expected: "SELECT * FROM users WHERE id = 1",
		},
		{
			name: "query with empty lines",
			input: `SELECT *

FROM users

WHERE id = 1`,
			expected: "SELECT * FROM users WHERE id = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.cleanSQL(tt.input)
			if result != tt.expected {
				t.Errorf("cleanSQL() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSQLParser_extractOnDuplicateKeyUpdate(t *testing.T) {
	parser := NewSQLParser()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no on duplicate key update",
			input:    "INSERT INTO users (id, name) VALUES (?, ?)",
			expected: "",
		},
		{
			name:     "with on duplicate key update",
			input:    "INSERT INTO users (id, name) VALUES (?, ?) ON DUPLICATE KEY UPDATE name = VALUES(name)",
			expected: "ON DUPLICATE KEY UPDATE name = VALUES(name)",
		},
		{
			name: "multiline on duplicate key update",
			input: `INSERT INTO users (id, name) VALUES (?, ?) 
ON DUPLICATE KEY UPDATE 
    name = VALUES(name),
    updated_at = NOW()`,
			expected: `ON DUPLICATE KEY UPDATE 
    name = VALUES(name),
    updated_at = NOW()`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.extractOnDuplicateKeyUpdate(tt.input)
			if result != tt.expected {
				t.Errorf("extractOnDuplicateKeyUpdate() = %q, want %q", result, tt.expected)
			}
		})
	}
}
