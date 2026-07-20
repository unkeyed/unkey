package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// SecurityFilter represents a row-level security constraint
type SecurityFilter struct {
	Column        string   // Column name
	AllowedValues []string // Values user is allowed to access
}

// Config for the parser
type Config struct {
	WorkspaceID            string
	TableAliases           map[string]string
	AllowedTables          []string
	PublicTableAliasesOnly bool             // Require table references to use a key from TableAliases.
	DisallowJoins          bool             // Reject FROM clauses containing more than one source.
	SecurityFilters        []SecurityFilter // Row-level security filters (auto-injected)
	Limit                  int
	QueryRangeDaysMax      int32 // Maximum historical data range user can query in days
}

// Parser rewrites ClickHouse queries
type Parser struct {
	config   Config
	stmt     *clickhouse.SelectQuery
	cteNames map[string]bool // Tracks CTE names defined in WITH clause
}
