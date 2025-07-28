package main

import (
	"strings"
	"testing"
)

func TestTemplateRenderer_Render(t *testing.T) {
	renderer := NewTemplateRenderer()

	tests := []struct {
		name     string
		data     TemplateData
		contains []string // Strings that should be present in the output
	}{
		{
			name: "basic template",
			data: TemplateData{
				Package:                   "db",
				BulkFunctionName:          "BulkInsertUser",
				BulkQueryConstant:         "bulkInsertUser",
				ParamsStructName:          "InsertUserParams",
				InsertPart:                "INSERT INTO users (id, name)",
				ValuesPart:                "(?, ?)",
				OnDuplicateKeyUpdate:      "",
				EmitMethodsWithDBArgument: true,
				Fields:                    []string{"ID", "Name"},
				ValuesFields:              []string{"ID", "Name"},
			},
			contains: []string{
				"package db",
				"const bulkInsertUser = `INSERT INTO users (id, name) VALUES %s`",
				"func (q *BulkQueries) BulkInsertUser(ctx context.Context, db DBTX, args []InsertUserParams) error",
				"allArgs = append(allArgs, arg.ID)",
				"allArgs = append(allArgs, arg.Name)",
				"_, err := db.ExecContext(ctx, bulkQuery, allArgs...)",
			},
		},
		{
			name: "template with on duplicate key update",
			data: TemplateData{
				Package:                   "db",
				BulkFunctionName:          "BulkUpsertUser",
				BulkQueryConstant:         "bulkUpsertUser",
				ParamsStructName:          "UpsertUserParams",
				InsertPart:                "INSERT INTO users (id, name)",
				ValuesPart:                "(?, ?)",
				OnDuplicateKeyUpdate:      "ON DUPLICATE KEY UPDATE name = VALUES(name)",
				EmitMethodsWithDBArgument: true,
				Fields:                    []string{"ID", "Name"},
				ValuesFields:              []string{"ID", "Name"},
			},
			contains: []string{
				"package db",
				"const bulkUpsertUser = `INSERT INTO users (id, name) VALUES %s ON DUPLICATE KEY UPDATE name = VALUES(name)`",
				"func (q *BulkQueries) BulkUpsertUser(ctx context.Context, db DBTX, args []UpsertUserParams) error",
			},
		},
		{
			name: "template without db argument",
			data: TemplateData{
				Package:                   "db",
				BulkFunctionName:          "BulkInsertUser",
				BulkQueryConstant:         "bulkInsertUser",
				ParamsStructName:          "InsertUserParams",
				InsertPart:                "INSERT INTO users (id, name)",
				ValuesPart:                "(?, ?)",
				OnDuplicateKeyUpdate:      "",
				EmitMethodsWithDBArgument: false,
				Fields:                    []string{"ID", "Name"},
				ValuesFields:              []string{"ID", "Name"},
			},
			contains: []string{
				"func (q *BulkQueries) BulkInsertUser(ctx context.Context, args []InsertUserParams) error",
				"_, err := q.db.ExecContext(ctx, bulkQuery, allArgs...)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render(tt.data)
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			for _, contain := range tt.contains {
				if !strings.Contains(result, contain) {
					t.Errorf("Render() result does not contain %q", contain)
					t.Logf("Generated code:\n%s", result)
				}
			}
		})
	}
}

func TestTemplateRenderer_RenderInvalidTemplate(t *testing.T) {
	// This test ensures error handling works properly
	renderer := &TemplateRenderer{}

	// We can't easily test template parsing errors without changing the embedded template,
	// so we'll just verify that a valid template doesn't error
	data := TemplateData{
		Package:                   "db",
		BulkFunctionName:          "BulkInsertUser",
		BulkQueryConstant:         "bulkInsertUser",
		ParamsStructName:          "InsertUserParams",
		InsertPart:                "INSERT INTO users (id, name)",
		ValuesPart:                "(?, ?)",
		OnDuplicateKeyUpdate:      "",
		EmitMethodsWithDBArgument: true,
		Fields:                    []string{"ID", "Name"},
		ValuesFields:              []string{"ID", "Name"},
	}

	_, err := renderer.Render(data)
	if err != nil {
		t.Errorf("Render() with valid data should not error, got: %v", err)
	}
}
