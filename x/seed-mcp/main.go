// Command seed-mcp is an MCP server for introspecting and seeding the Unkey
// MySQL database. It speaks MCP over stdio and is intended to be launched as a
// subprocess by an MCP client (e.g. Claude Code) during local development.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/unkeyed/unkey/pkg/base58"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
)

const (
	appName    = "unkey-seed-mcp"
	appVersion = "0.1.0"

	// dsn points at the local MySQL from web/apps/dashboard/dev/docker-compose.yaml.
	// This MCP is local-dev only; if you ever need to point it elsewhere, change
	// the compose, not this constant.
	dsn = "unkey:password@tcp(127.0.0.1:3306)/unkey?parseTime=true&interpolateParams=true"
)

// tablePrefix maps a MySQL table name to the uid prefix used for its `id`
// column. Tables whose schema has no `id` column (junction tables and
// encrypted_keys) are intentionally absent: maybeGenerateID skips id
// generation when the table is not in this map.
var tablePrefix = map[string]uid.Prefix{
	"workspaces":           uid.WorkspacePrefix,
	"apis":                 uid.APIPrefix,
	"keys":                 uid.KeyPrefix,
	"key_auth":             uid.KeySpacePrefix,
	"identities":           uid.IdentityPrefix,
	"ratelimits":           uid.RatelimitPrefix,
	"ratelimit_namespaces": uid.RatelimitNamespacePrefix,
	"ratelimit_overrides":  uid.RatelimitOverridePrefix,
	"permissions":          uid.PermissionPrefix,
	"roles":                uid.RolePrefix,
	"audit_log":            uid.AuditLogPrefix,
	"audit_log_target":     uid.AuditLogPrefix,
	"projects":             uid.ProjectPrefix,
	"environments":         uid.EnvironmentPrefix,
	"apps":                 uid.AppPrefix,
	"deployments":          uid.DeploymentPrefix,
	"custom_domains":       uid.DomainPrefix,
	"openapi_specs":        uid.OpenApiSpecPrefix,
	"certificates":         uid.CertificatePrefix,
	"frontline_routes":     uid.FrontlineRoutePrefix,
	"regions":              uid.RegionPrefix,
	"clusters":             uid.ClusterPrefix,
	"instances":            uid.InstancePrefix,
	"sentinels":            uid.SentinelPrefix,
	"key_migrations":       uid.KeyPrefix,
}

func main() {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := db.Ping(); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	s := mcp.NewServer(&mcp.Implementation{Name: appName, Version: appVersion}, nil) //nolint:exhaustruct

	registerListTables(s, db)
	registerSampleRows(s, db)
	registerQuery(s, db)
	registerGenerateKey(s)
	registerInsert(s, db)
	registerBulkInsert(s, db)

	log.Printf("%s v%s starting", appName, appVersion)
	if err := s.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// ---------- list_tables ----------

type listTablesIn struct{}

type listTablesOut struct {
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
}

func registerListTables(s *mcp.Server, db *sql.DB) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "list_tables",
		Description: "List all tables in the connected database.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listTablesIn) (*mcp.CallToolResult, listTablesOut, error) {
		var dbName string
		if err := db.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&dbName); err != nil {
			return nil, listTablesOut{}, err
		}
		rows, err := db.QueryContext(ctx,
			`SELECT table_name FROM information_schema.tables
			 WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'
			 ORDER BY table_name`)
		if err != nil {
			return nil, listTablesOut{}, err
		}
		defer func() { _ = rows.Close() }()

		var tables []string
		for rows.Next() {
			var t string
			if err := rows.Scan(&t); err != nil {
				return nil, listTablesOut{}, err
			}
			tables = append(tables, t)
		}
		out := listTablesOut{Database: dbName, Tables: tables}
		return textResult(fmt.Sprintf("%d tables in database %q", len(tables), dbName)), out, nil
	})
}

// ---------- sample_rows ----------

type sampleRowsIn struct {
	Table string `json:"table" jsonschema:"the table to sample"`
	Limit int    `json:"limit,omitempty" jsonschema:"max rows to return (default 5, capped at 50)"`
}

type sampleRowsOut struct {
	Table   string                   `json:"table"`
	Columns []string                 `json:"columns"`
	Rows    []map[string]any         `json:"rows"`
}

func registerSampleRows(s *mcp.Server, db *sql.DB) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "sample_rows",
		Description: "Return a few real rows from a table so you can match realistic value shapes when seeding.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in sampleRowsIn) (*mcp.CallToolResult, sampleRowsOut, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 5
		}
		if limit > 50 {
			limit = 50
		}
		// Table name is interpolated, so guard against injection by allowing only
		// tables that exist in information_schema.
		if err := assertTableExists(ctx, db, in.Table); err != nil {
			return nil, sampleRowsOut{}, err
		}
		q := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d", in.Table, limit)
		rows, err := db.QueryContext(ctx, q)
		if err != nil {
			return nil, sampleRowsOut{}, err
		}
		defer func() { _ = rows.Close() }()

		cols, err := rows.Columns()
		if err != nil {
			return nil, sampleRowsOut{}, err
		}
		out := sampleRowsOut{Table: in.Table, Columns: cols}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, sampleRowsOut{}, err
			}
			row := make(map[string]any, len(cols))
			for i, c := range cols {
				row[c] = normalizeSQLValue(vals[i])
			}
			out.Rows = append(out.Rows, row)
		}
		return textResult(fmt.Sprintf("%d rows from %s", len(out.Rows), in.Table)), out, nil
	})
}

// ---------- query (read-only SELECT/SHOW/DESCRIBE) ----------

type queryIn struct {
	SQL  string `json:"sql" jsonschema:"a single read-only statement: SELECT, SHOW, DESCRIBE, or EXPLAIN"`
	Args []any  `json:"args,omitempty" jsonschema:"positional ? parameters"`
}

type queryOut struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

func registerQuery(s *mcp.Server, db *sql.DB) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name:        "query",
		Description: "Run a read-only SQL statement (SELECT/SHOW/DESCRIBE/EXPLAIN). Use ? placeholders for args.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in queryIn) (*mcp.CallToolResult, queryOut, error) {
		if err := assertReadOnly(in.SQL); err != nil {
			return nil, queryOut{}, err
		}
		rows, err := db.QueryContext(ctx, in.SQL, in.Args...)
		if err != nil {
			return nil, queryOut{}, err
		}
		defer func() { _ = rows.Close() }()

		cols, err := rows.Columns()
		if err != nil {
			return nil, queryOut{}, err
		}
		out := queryOut{Columns: cols}
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				return nil, queryOut{}, err
			}
			row := make(map[string]any, len(cols))
			for i, c := range cols {
				row[c] = normalizeSQLValue(vals[i])
			}
			out.Rows = append(out.Rows, row)
		}
		return textResult(fmt.Sprintf("%d rows", len(out.Rows))), out, nil
	})
}

// ---------- insert ----------

type insertIn struct {
	Table string         `json:"table"`
	Row   map[string]any `json:"row" jsonschema:"column -> value map; the id column is auto-generated if omitted and a uid prefix is registered for the table"`
}

type insertOut struct {
	Table string `json:"table"`
	ID    string `json:"id,omitempty"` // generated or supplied; empty for tables without an `id` column
	PK    int64  `json:"pk,omitempty"` // last_insert_id when present
}

func registerInsert(s *mcp.Server, db *sql.DB) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name: "insert",
		Description: "Insert a single row. If the row omits `id` and the table has a registered uid prefix, " +
			"generates a prefixed id automatically. Returns the (possibly generated) `id` for use in foreign keys.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in insertIn) (*mcp.CallToolResult, insertOut, error) {
		if err := assertTableExists(ctx, db, in.Table); err != nil {
			return nil, insertOut{}, err
		}
		maybeGenerateID(in.Table, in.Row)

		cols, placeholders, args := buildInsert(in.Row)
		q := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", in.Table, cols, placeholders)
		res, err := db.ExecContext(ctx, q, args...)
		if err != nil {
			return nil, insertOut{}, fmt.Errorf("insert into %s failed: %w", in.Table, err)
		}
		out := insertOut{Table: in.Table, ID: "", PK: 0}
		if id, ok := in.Row["id"].(string); ok {
			out.ID = id
		}
		if pk, err := res.LastInsertId(); err == nil {
			out.PK = pk
		}
		msg := fmt.Sprintf("inserted into %s", in.Table)
		if out.ID != "" {
			msg += " id=" + out.ID
		}
		return textResult(msg), out, nil
	})
}

// ---------- bulk_insert ----------

type bulkInsertIn struct {
	Table string           `json:"table"`
	Rows  []map[string]any `json:"rows" jsonschema:"each row is column->value; the id column is auto-generated per row if omitted and a prefix is registered"`
}

type bulkInsertOut struct {
	Table string   `json:"table"`
	Count int      `json:"count"`
	IDs   []string `json:"ids,omitempty"` // one per row in input order; empty entries for tables without `id`
}

func registerBulkInsert(s *mcp.Server, db *sql.DB) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name: "bulk_insert",
		Description: "Insert many rows of the same table in a single transaction, chunked into batches of 1000. " +
			"Auto-generates `id` per row when omitted and a uid prefix is registered. Returns ids in input order.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in bulkInsertIn) (*mcp.CallToolResult, bulkInsertOut, error) {
		if err := assertTableExists(ctx, db, in.Table); err != nil {
			return nil, bulkInsertOut{}, err
		}
		if len(in.Rows) == 0 {
			return textResult("nothing to insert"), bulkInsertOut{Table: in.Table, Count: 0, IDs: nil}, nil
		}

		// Generate ids first so they're stable regardless of batch boundaries
		// and so we can return them in the same order as the input.
		ids := make([]string, len(in.Rows))
		for i, row := range in.Rows {
			maybeGenerateID(in.Table, row)
			if id, ok := row["id"].(string); ok {
				ids[i] = id
			}
		}

		// A single multi-VALUES INSERT requires one fixed column list. Use the
		// union of keys across rows; rows that omit a key get SQL NULL via the
		// nil-arg fallback in the loop below. This matches the agent's mental
		// model of "missing column = unset = NULL".
		colSet := map[string]struct{}{}
		for _, row := range in.Rows {
			for k := range row {
				colSet[k] = struct{}{}
			}
		}
		cols := make([]string, 0, len(colSet))
		for k := range colSet {
			cols = append(cols, k)
		}
		sort.Strings(cols)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return nil, bulkInsertOut{}, err
		}
		defer func() { _ = tx.Rollback() }()

		const batch = 1000
		for start := 0; start < len(in.Rows); start += batch {
			end := start + batch
			if end > len(in.Rows) {
				end = len(in.Rows)
			}
			chunk := in.Rows[start:end]

			var b strings.Builder
			b.WriteString("INSERT INTO `")
			b.WriteString(in.Table)
			b.WriteString("` (")
			for i, c := range cols {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('`')
				b.WriteString(c)
				b.WriteByte('`')
			}
			b.WriteString(") VALUES ")

			args := make([]any, 0, len(chunk)*len(cols))
			for i, row := range chunk {
				if i > 0 {
					b.WriteByte(',')
				}
				b.WriteByte('(')
				for j, c := range cols {
					if j > 0 {
						b.WriteByte(',')
					}
					b.WriteByte('?')
					args = append(args, row[c]) // missing keys → nil → SQL NULL
				}
				b.WriteByte(')')
			}

			if _, err := tx.ExecContext(ctx, b.String(), args...); err != nil {
				return nil, bulkInsertOut{}, fmt.Errorf("bulk insert into %s failed at row %d: %w", in.Table, start, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, bulkInsertOut{}, err
		}

		out := bulkInsertOut{Table: in.Table, Count: len(in.Rows), IDs: ids}
		return textResult(fmt.Sprintf("inserted %d rows into %s", out.Count, in.Table)), out, nil
	})
}

// ---------- generate_key ----------

type generateKeyIn struct {
	Prefix     string `json:"prefix,omitempty" jsonschema:"key prefix (e.g. 'sk', 'unkey'); empty for no prefix"`
	ByteLength int    `json:"byte_length,omitempty" jsonschema:"random byte length (16-255, default 16)"`
}

type generateKeyOut struct {
	ID    string `json:"id"`    // key_xxx id for the keys.id column
	Key   string `json:"key"`   // plaintext key (the secret to hand to the API caller)
	Hash  string `json:"hash"`  // sha256 of the plaintext, for the keys.hash column
	Start string `json:"start"` // first 4 chars of the body, for the keys.start column
}

func registerGenerateKey(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{ //nolint:exhaustruct
		Name: "generate_key",
		Description: "Generate a new Unkey API key. Returns id, plaintext key, sha256 hash, and start. " +
			"Use the returned hash/start when inserting into the `keys` table; hand the plaintext key to the API caller.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, in generateKeyIn) (*mcp.CallToolResult, generateKeyOut, error) {
		n := in.ByteLength
		if n == 0 {
			n = 16
		}
		if n < 16 || n > 255 {
			return nil, generateKeyOut{}, fmt.Errorf("byte_length must be between 16 and 255 (got %d)", n)
		}

		buf := make([]byte, n)
		if _, err := rand.Read(buf); err != nil {
			return nil, generateKeyOut{}, fmt.Errorf("read random bytes: %w", err)
		}

		encoded := base58.Encode(buf)
		fullKey := encoded
		start := encoded[:4]
		if in.Prefix != "" {
			fullKey = in.Prefix + "_" + encoded
			start = in.Prefix + "_" + encoded[:4]
		}

		out := generateKeyOut{
			ID:    uid.New(uid.KeyPrefix),
			Key:   fullKey,
			Hash:  hash.Sha256(fullKey),
			Start: start,
		}
		return textResult("generated key " + out.ID), out, nil
	})
}

// ---------- helpers ----------

func textResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{ //nolint:exhaustruct
		Content: []mcp.Content{&mcp.TextContent{Text: msg}}, //nolint:exhaustruct
	}
}

func maybeGenerateID(table string, row map[string]any) {
	if _, has := row["id"]; has {
		return
	}
	prefix, ok := tablePrefix[table]
	if !ok {
		return
	}
	row["id"] = uid.New(prefix)
}

func buildInsert(row map[string]any) (cols string, placeholders string, args []any) {
	keys := make([]string, 0, len(row))
	for k := range row {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var c, p strings.Builder
	args = make([]any, 0, len(keys))
	for i, k := range keys {
		if i > 0 {
			c.WriteByte(',')
			p.WriteByte(',')
		}
		c.WriteByte('`')
		c.WriteString(k)
		c.WriteByte('`')
		p.WriteByte('?')
		args = append(args, row[k])
	}
	return c.String(), p.String(), args
}

func assertReadOnly(q string) error {
	trimmed := strings.TrimSpace(strings.ToUpper(q))
	for _, prefix := range []string{"SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH"} {
		if strings.HasPrefix(trimmed, prefix+" ") || trimmed == prefix {
			return nil
		}
	}
	return fmt.Errorf("query tool only accepts SELECT/SHOW/DESCRIBE/EXPLAIN/WITH; use insert or bulk_insert for writes")
}

func assertTableExists(ctx context.Context, db *sql.DB, table string) error {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = ?`, table).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("table %q does not exist in current database", table)
	}
	return nil
}

// normalizeSQLValue makes scanned values JSON-friendly. database/sql returns
// []byte for many MySQL types; the agent reads strings more easily.
func normalizeSQLValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	default:
		return v
	}
}
