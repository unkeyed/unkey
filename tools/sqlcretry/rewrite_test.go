package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRewrite(t *testing.T) {
	for _, tt := range []struct {
		name       string
		database   string
		parameters string
		arguments  string
	}{
		{name: "database argument", database: "db", parameters: "ctx context.Context, db DBTX, id string", arguments: "ctx, db, id"},
		{name: "bound database", database: "q.db", parameters: "ctx context.Context, id string", arguments: "ctx, id"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source := fmt.Sprintf(`package db
import "context"
const findUser = "-- name: FindUser :one\nSELECT id FROM users WHERE id = ?"
// FindUser reads a user.
func (q *Queries) FindUser(%s) (string, error) {
    var result string
    err := %s.QueryRowContext(ctx, findUser, id).Scan(&result)
    return result, err
}
`, tt.parameters, tt.database)
			output, err := rewrite([]byte(source))
			require.NoError(t, err)
			_, err = parser.ParseFile(token.NewFileSet(), "", output, parser.AllErrors)
			require.NoError(t, err)
			require.Contains(t, string(output), fmt.Sprintf("func (q *Queries) FindUser(%s) (string, error)", tt.parameters))
			require.Contains(t, string(output), "mysql.ReadWithRetry(ctx, "+tt.database+", func() (string, error)")
			require.Contains(t, string(output), "// FindUser reads a user.\nfunc (q *Queries) FindUser")
			require.Contains(t, string(output), "return q.findUser("+tt.arguments+")")

			again, err := rewrite(output)
			require.NoError(t, err)
			require.Equal(t, output, again)
		})
	}
}

func TestRewriteLeavesUnsafeQueriesUnchanged(t *testing.T) {
	for _, query := range []string{
		"SELECT id FROM users FOR UPDATE",
		"SELECT GET_LOCK('name', 1)",
		"SELECT send_email(id) FROM users",
		"UPDATE users SET enabled = true",
	} {
		source := fmt.Sprintf(`package db
import "context"
const operation = %q
func (q *Queries) Operation(ctx context.Context, db DBTX) (string, error) {
    var result string
    err := db.QueryRowContext(ctx, operation).Scan(&result)
    return result, err
}
`, "-- name: Operation :one\n"+query)
		output, err := rewrite([]byte(source))
		require.NoError(t, err)
		require.Equal(t, source, string(output))
	}
}

func TestRewriteHandlesSQLConcatenation(t *testing.T) {
	source := []byte("package db\nimport \"context\"\nconst read = `-- name: Read :one\nSELECT id FROM ` + \"`users`\" + ` WHERE id = ?`\nfunc (q *Queries) Read(ctx context.Context, db DBTX, id string) (string, error) { return id, nil }\n")
	output, err := rewrite(source)
	require.NoError(t, err)
	require.Contains(t, string(output), "mysql.ReadWithRetry")
}

func TestRewriteRejectsMissingMethod(t *testing.T) {
	_, err := rewrite([]byte("package db\nconst read = `-- name: Read :one\nSELECT id FROM users`\n"))
	require.ErrorContains(t, err, "Read: generated query method not found")
}
