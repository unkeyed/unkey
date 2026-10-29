package main

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/stretchr/testify/require"
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
			expected: 2,
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
			expected: 2,
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
						Params:   []*plugin.Parameter{},
						Filename: "insert_default_user.sql",
					},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := generator.Generate(tt.req)
			require.NoError(t, err)

			require.Len(t, resp.GetFiles(), tt.expected)

			for _, file := range resp.GetFiles() {
				require.NotEmpty(t, file.GetName())
				require.NotEmpty(t, file.GetContents())
			}
		})
	}
}

func TestGenerator_Configure(t *testing.T) {
	t.Run("preserves defaults without options", func(t *testing.T) {
		generator := NewGenerator()

		require.NoError(t, generator.Configure(nil))

		require.Equal(t, "db", generator.options.Package)
		require.True(t, generator.options.EmitMethodsWithDBArgument)
	})

	t.Run("applies explicit options", func(t *testing.T) {
		generator := NewGenerator()

		err := generator.Configure([]byte(`{"package":"ctrl_db","emit_methods_with_db_argument":false}`))
		require.NoError(t, err)

		require.Equal(t, "ctrl_db", generator.options.Package)
		require.False(t, generator.options.EmitMethodsWithDBArgument)
	})

	t.Run("rejects invalid JSON", func(t *testing.T) {
		generator := NewGenerator()

		require.Error(t, generator.Configure([]byte(`{`)))
	})
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
			require.Equal(t, tt.expected, result)
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
			require.Equal(t, tt.expected, result)
		})
	}
}
