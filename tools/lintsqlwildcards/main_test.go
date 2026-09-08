package main

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScannerRejectsAllColumnQueries protects every production query syntax
// covered by the repository-wide lint task.
func TestScannerRejectsAllColumnQueries(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		source   string
		wantKind string
	}{
		{
			name: "explicit SQL projection", path: "query.sql",
			source: "SELECT id, name FROM users;\n",
		},
		{
			name: "SQL aggregate star", path: "query.sql",
			source: "SELECT COUNT(*) FROM users;\n",
		},
		{
			name: "SQL comments, literals, and permission targets", path: "query.sql",
			source: "-- SELECT * FROM ignored_comment\nSELECT 'SELECT * FROM ignored_literal' AS example;\nGRANT SELECT ON database.* TO user;\n",
		},
		{
			name: "Go comments and non-SQL prose", path: "main.go",
			source: "package main\n\n// SELECT * FROM comments must not count.\nconst guidance = \"Avoid wildcard SQL projections\"\n",
		},
		{
			name: "explicit Drizzle projection", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: { id: true } });\n",
		},
		{
			name: "explicit nested Drizzle projection", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: { id: true }, with: { memberships: { columns: { id: true } } } });\n",
		},
		{
			name: "disabled Drizzle relation", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: { id: true }, with: { memberships: false } });\n",
		},
		{
			name: "explicit query-builder projection", path: "query.ts",
			source: "db.select({ id: users.id }).from(users);\n",
		},
		{
			name: "Drizzle generic return type", path: "web/apps/dashboard/query.ts",
			source: "let row: Awaited<ReturnType<typeof db.query.users.findFirst<typeof projection>>>;\n",
		},
		{
			name: "immutable migration statement", path: "pkg/clickhouse/migrations/20260419000000.sql",
			source: "SELECT * FROM default.instance_checkpoints_v1 FINAL;\n",
		},
		{
			name: "SQL wildcard", path: "query.sql",
			source: "SELECT * FROM users;\n", wantKind: "SELECT wildcard",
		},
		{
			name: "SQL wildcard with modifier", path: "query.sql",
			source: "SELECT DISTINCT * FROM users;\n", wantKind: "SELECT wildcard",
		},
		{
			name: "SQL wildcard after another column", path: "query.sql",
			source: "SELECT id, * FROM users;\n", wantKind: "SELECT wildcard",
		},
		{
			name: "qualified SQL wildcard", path: "query.sql",
			source: "SELECT users.* FROM users;\n", wantKind: "qualified wildcard selection",
		},
		{
			name: "sqlc embed", path: "query.sql",
			source: "SELECT sqlc.embed(users) FROM users;\n", wantKind: "sqlc.embed all-column selection",
		},
		{
			name: "Go raw SQL wildcard", path: "main.go",
			source: "package main\n\nconst query = `SELECT * FROM users`\n", wantKind: "SELECT wildcard",
		},
		{
			name: "TypeScript raw SQL wildcard", path: "query.ts",
			source: "const query = \"SELECT * FROM users\";\n", wantKind: "SELECT wildcard",
		},
		{
			name: "Drizzle query without columns", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ where: eq(users.id, id) });\n", wantKind: "missing Drizzle query columns projection",
		},
		{
			name: "Drizzle all-column relation", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({\n  columns: { id: true },\n  with: { memberships: true },\n});\n", wantKind: "implicit all-column Drizzle relation",
		},
		{
			name: "Drizzle negative projection", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: { secret: false } });\n", wantKind: "non-positive Drizzle columns projection",
		},
		{
			name: "empty Drizzle columns", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: {} });\n", wantKind: "empty Drizzle columns projection",
		},
		{
			name: "dynamic Drizzle columns", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns });\n", wantKind: "missing Drizzle query columns projection",
		},
		{
			name: "nonliteral Drizzle config", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany(options);\n", wantKind: "nonliteral Drizzle query config",
		},
		{
			name: "generic Drizzle query", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany<{ columns: { id: true } }>({ columns: { id: true } });\n", wantKind: "generic Drizzle relational query",
		},
		{
			name: "Drizzle relation without columns", path: "web/apps/dashboard/query.ts",
			source: "db.query.users.findMany({ columns: { id: true }, with: { memberships: { where: active } } });\n", wantKind: "missing Drizzle relation columns projection",
		},
		{
			name: "query-builder selectAll", path: "query.ts",
			source: "db.selectFrom(\"users\").selectAll();\n", wantKind: "query-builder selectAll selection",
		},
		{
			name: "query-builder wildcard string", path: "query.ts",
			source: "db.selectFrom(\"users\").select(\"*\");\n", wantKind: "query-builder string wildcard selection",
		},
		{
			name: "query-builder qualified wildcard array", path: "query.ts",
			source: "db.selectFrom(\"users\").select([\"users.*\"]);\n", wantKind: "query-builder string wildcard selection",
		},
		{
			name: "empty Drizzle select", path: "query.ts",
			source: "db.select().from(users);\n", wantKind: "empty query-builder .select()",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			r := reporter{stderr: &output, violations: 0}
			r.scanFile(test.path, []byte(test.source))
			if test.wantKind == "" {
				require.Zero(t, r.violations, "scanFile(%q) output:\n%s", test.path, output.String())
				return
			}
			require.NotZero(t, r.violations, "scanFile(%q) did not reject source", test.path)
			require.Contains(t, output.String(), "forbidden "+test.wantKind)
		})
	}
}

// TestIsExemptTestPathLimitsExemptions keeps wildcard examples confined to test and
// fixture paths.
func TestIsExemptTestPathLimitsExemptions(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "query.sql"},
		{path: "query_test.sql", want: true},
		{path: "src/query.test.ts", want: true},
		{path: "src/query.spec.tsx", want: true},
		{path: "pkg/testdata/query.sql", want: true},
		{path: "web/__tests__/query.ts", want: true},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			require.Equal(t, test.want, isExemptTestPath(test.path), "isExemptTestPath(%q)", test.path)
		})
	}
}

func BenchmarkScanner(b *testing.B) {
	source := []byte(`db.query.users.findMany({
  columns: { id: true, name: true },
  with: { memberships: { columns: { id: true } } },
});`)
	for b.Loop() {
		r := reporter{stderr: io.Discard, violations: 0}
		r.scanFile("web/apps/dashboard/query.ts", source)
	}
}
