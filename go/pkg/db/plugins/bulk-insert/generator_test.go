package main

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestGenerator_Generate(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name     string
		req      *plugin.GenerateRequest
		expected int // expected number of generated files
	}{
		{
			name: "single insert query",
			req: &plugin.GenerateRequest{
				Queries: []*plugin.Query{
					{
						Name: "InsertUser",
						Text: "INSERT INTO users (id, name) VALUES (?, ?)",
						InsertIntoTable: &plugin.Identifier{
							Name: "users",
						},
						Params: []*plugin.Parameter{
							{
								Column: &plugin.Column{
									Name: "id",
								},
							},
							{
								Column: &plugin.Column{
									Name: "name",
								},
							},
						},
						Filename: "insert_user.sql",
					},
				},
			},
			expected: 2, // 1 bulk function + 1 interface file
		},
		{
			name: "mixed query types",
			req: &plugin.GenerateRequest{
				Queries: []*plugin.Query{
					{
						Name: "InsertUser",
						Text: "INSERT INTO users (id, name) VALUES (?, ?)",
						InsertIntoTable: &plugin.Identifier{
							Name: "users",
						},
						Params: []*plugin.Parameter{
							{
								Column: &plugin.Column{
									Name: "id",
								},
							},
						},
						Filename: "insert_user.sql",
					},
					{
						Name: "GetUser",
						Text: "SELECT id, name FROM users WHERE id = ?",
						// No InsertIntoTable - should be skipped
						Params: []*plugin.Parameter{
							{
								Column: &plugin.Column{
									Name: "id",
								},
							},
						},
						Filename: "get_user.sql",
					},
				},
			},
			expected: 2, // 1 bulk function + 1 interface file (only insert query generates files)
		},
		{
			name: "insert query without parameters",
			req: &plugin.GenerateRequest{
				Queries: []*plugin.Query{
					{
						Name: "InsertDefaultUser",
						Text: "INSERT INTO users (id, name) VALUES (DEFAULT, 'default')",
						InsertIntoTable: &plugin.Identifier{
							Name: "users",
						},
						Params:   []*plugin.Parameter{}, // No parameters
						Filename: "insert_default_user.sql",
					},
				},
			},
			expected: 0, // Should be skipped due to no parameters
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := generator.Generate(tt.req)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			if len(resp.GetFiles()) != tt.expected {
				t.Errorf("Generate() returned %d files, want %d", len(resp.GetFiles()), tt.expected)
			}

			// Verify file names are correctly generated
			for _, file := range resp.GetFiles() {
				if file.GetName() == "" {
					t.Error("Generated file has empty name")
				}
				if len(file.GetContents()) == 0 {
					t.Error("Generated file has empty contents")
				}
			}
		})
	}
}

func TestGenerator_isInsertQuery(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name     string
		query    *plugin.Query
		expected bool
	}{
		{
			name: "insert query with parameters",
			query: &plugin.Query{
				InsertIntoTable: &plugin.Identifier{Name: "users"},
				Params:          []*plugin.Parameter{{Column: &plugin.Column{Name: "id"}}},
			},
			expected: true,
		},
		{
			name: "insert query without parameters",
			query: &plugin.Query{
				InsertIntoTable: &plugin.Identifier{Name: "users"},
				Params:          []*plugin.Parameter{},
			},
			expected: false,
		},
		{
			name: "non-insert query",
			query: &plugin.Query{
				InsertIntoTable: nil,
				Params:          []*plugin.Parameter{{Column: &plugin.Column{Name: "id"}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.isInsertQuery(tt.query)
			if result != tt.expected {
				t.Errorf("isInsertQuery() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerator_extractFieldNames(t *testing.T) {
	generator := NewGenerator()

	tests := []struct {
		name     string
		params   []*plugin.Parameter
		expected []string
	}{
		{
			name: "simple parameters",
			params: []*plugin.Parameter{
				{Column: &plugin.Column{Name: "id"}},
				{Column: &plugin.Column{Name: "name"}},
				{Column: &plugin.Column{Name: "created_at"}},
			},
			expected: []string{"ID", "Name", "CreatedAt"},
		},
		{
			name: "parameters with snake_case",
			params: []*plugin.Parameter{
				{Column: &plugin.Column{Name: "user_id"}},
				{Column: &plugin.Column{Name: "git_repository_url"}},
			},
			expected: []string{"UserID", "GitRepositoryUrl"},
		},
		{
			name: "parameters with nil column",
			params: []*plugin.Parameter{
				{Column: &plugin.Column{Name: "id"}},
				{Column: nil}, // Should be skipped
				{Column: &plugin.Column{Name: "name"}},
			},
			expected: []string{"ID", "Name"},
		},
		{
			name:     "no parameters",
			params:   []*plugin.Parameter{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.extractFieldNames(tt.params)
			if len(result) != len(tt.expected) {
				t.Errorf("extractFieldNames() returned %d fields, want %d", len(result), len(tt.expected))
			}
			for i, field := range result {
				if i < len(tt.expected) && field != tt.expected[i] {
					t.Errorf("extractFieldNames()[%d] = %q, want %q", i, field, tt.expected[i])
				}
			}
		})
	}
}
