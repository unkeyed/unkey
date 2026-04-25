package urql

import (
	"context"
	"errors"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ErrNotURQL is returned by Compile when the input query references no URQL
// logical tables. Callers should fall back to handling the input as raw
// ClickHouse SQL.
var ErrNotURQL = errors.New("query is not URQL")

// Compile takes a URQL query string and compiles it to ClickHouse SQL plus a
// map of column-format hints.
//
// The fallback rule: if the query references no URQL logical table, returns
// ErrNotURQL and the caller should treat the input as raw ClickHouse SQL.
// If a URQL logical table is referenced and compilation fails for any other
// reason (unknown column, mixed legacy/URQL tables, etc.), URQL owns the
// query and surfaces its error to the user.
//
// The returned SQL is intended to be passed through pkg/clickhouse/query-parser
// for the security pass (workspace isolation, RBAC, time-range bounds,
// function allowlist). URQL never bypasses that layer.
func Compile(ctx context.Context, query string, schema *Schema) (string, map[string]string, error) {
	parser := clickhouse.NewParser(query)
	stmts, err := parser.ParseStmts()
	if err != nil {
		// If the input doesn't even parse, URQL has no claim on it — let
		// the legacy parser produce its own (more familiar) syntax error.
		return "", nil, ErrNotURQL
	}
	if len(stmts) == 0 {
		return "", nil, ErrNotURQL
	}

	stmt, ok := stmts[0].(*clickhouse.SelectQuery)
	if !ok {
		// URQL only operates on SELECTs. Anything else (INSERT/UPDATE/DDL)
		// gets handed to the legacy parser, which rejects it.
		return "", nil, ErrNotURQL
	}

	c := &compiler{
		stmt:          stmt,
		schema:        schema,
		cteNames:      make(map[string]bool),
		table:         nil,
		useTimeBucket: false,
		granularity:   GranularityRaw,
	}

	c.collectCTENames()

	owned, err := c.detectOwnership()
	if err != nil {
		return "", nil, err
	}
	if !owned {
		return "", nil, ErrNotURQL
	}

	if err := c.validateAllTablesAreLogical(); err != nil {
		return "", nil, err
	}

	c.detectTimeBucket()

	if err := c.resolveVariant(); err != nil {
		return "", nil, err
	}

	if err := c.rewriteTables(); err != nil {
		return "", nil, err
	}

	if err := c.validateTimeBucketArgs(); err != nil {
		return "", nil, err
	}

	columnFormats, err := c.extractPrettyFormat()
	if err != nil {
		return "", nil, err
	}

	if err := c.validateColumnReferences(); err != nil {
		return "", nil, err
	}

	if err := c.validateAllowedValues(); err != nil {
		return "", nil, err
	}

	sql := c.stmt.String()
	sql = replaceVirtualColumns(sql, c.table, c.granularity)
	if c.useTimeBucket {
		sql = replaceTimeBucket(sql, c.table.TimeColumn)
	}

	return sql, columnFormats, nil
}

// compiler holds the per-Compile state. It carries the parsed statement, the
// schema, and the resolved logical table + variant once detected.
type compiler struct {
	stmt   *clickhouse.SelectQuery
	schema *Schema

	cteNames map[string]bool

	// table is the single URQL logical table referenced by this query.
	// Set during detectOwnership. Phase 1 only supports one logical table
	// per query (multiple references to the same table are fine).
	table *LogicalTable

	// useTimeBucket is true if the query contains a timeBucket() call.
	useTimeBucket bool

	// granularity is the resolved physical variant of `table`. Set during
	// resolveVariant. All occurrences of the logical table in the query
	// resolve to this same variant.
	granularity Granularity
}

func urqlError(msg, public string) error {
	return fault.New(msg,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(public),
	)
}

func urqlErrorf(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return urqlError(msg, msg)
}
