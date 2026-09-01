package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGeneratedFileUpToDate fails when columns_generated.go drifts from the
// schema structs. Fix by running `mise run generate` (or `go generate ./...`).
func TestGeneratedFileUpToDate(t *testing.T) {
	want, err := generate("..")
	require.NoError(t, err)

	got, err := os.ReadFile("../" + outputFile)
	require.NoError(t, err)
	require.Equal(t, want, string(got),
		"columns_generated.go is stale, run `mise run generate`")
}

func parseType(t *testing.T, src string) (*ast.GenDecl, *ast.TypeSpec, *ast.StructType) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "src.go", "package p\n\n"+src,
		parser.ParseComments|parser.SkipObjectResolution)
	require.NoError(t, err)
	genDecl, ok := file.Decls[0].(*ast.GenDecl)
	require.True(t, ok)
	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	require.True(t, ok)
	structType, ok := typeSpec.Type.(*ast.StructType)
	require.True(t, ok)
	return genDecl, typeSpec, structType
}

func TestDeriveColumns(t *testing.T) {
	t.Run("ch tag wins, dash and unexported skipped", func(t *testing.T) {
		_, _, structType := parseType(t, `
type row struct {
	RequestID string `+"`ch:\"request_id\"`"+`
	Disabled  string `+"`ch:\"-\"`"+`
	NoTag     string
	hidden    string
}`)
		columns, err := deriveColumns(structType)
		require.NoError(t, err)
		require.Equal(t, []string{"request_id", "NoTag"}, columns)
	})

	t.Run("embedded fields are rejected", func(t *testing.T) {
		_, _, structType := parseType(t, `
type row struct {
	base
	ID string `+"`ch:\"id\"`"+`
}`)
		_, err := deriveColumns(structType)
		require.Error(t, err)
	})
}

func TestTableName(t *testing.T) {
	t.Run("directive is extracted", func(t *testing.T) {
		genDecl, typeSpec, _ := parseType(t, `
// row is a table row.
//
//unkey:table default.rows_v1
type row struct {
	ID string `+"`ch:\"id\"`"+`
}`)
		table, err := tableName(genDecl, typeSpec)
		require.NoError(t, err)
		require.Equal(t, "default.rows_v1", table)
	})

	t.Run("no directive means not a row", func(t *testing.T) {
		genDecl, typeSpec, _ := parseType(t, `
// payload is not inserted directly.
type payload struct {
	Image string `+"`json:\"image\"`"+`
}`)
		table, err := tableName(genDecl, typeSpec)
		require.NoError(t, err)
		require.Empty(t, table)
	})

	t.Run("malformed directive errors", func(t *testing.T) {
		genDecl, typeSpec, _ := parseType(t, `
//unkey:table
type row struct {
	ID string `+"`ch:\"id\"`"+`
}`)
		_, err := tableName(genDecl, typeSpec)
		require.Error(t, err)
	})
}
