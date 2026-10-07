package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// SecurityFilter represents a row-level security constraint
type SecurityFilter struct {
	Column        string   // Column name
	AllowedValues []string // Values user is allowed to access
}

// SecurityScope is one allowed row scope. Filters within a scope are combined
// with AND, while Config.SecurityScopes combines scopes with OR.
type SecurityScope struct {
	Filters []SecurityFilter
}

// Config for the parser
type Config struct {
	WorkspaceID       string
	TableAliases      map[string]string
	AllowedTables     []string
	SecurityFilters   []SecurityFilter // Row-level security filters (auto-injected)
	SecurityScopes    []SecurityScope  // nil permits all workspace rows; non-nil empty denies all
	Limit             int
	QueryRangeDaysMax int32 // Maximum historical data range user can query in days
}

// Parser rewrites ClickHouse queries
type Parser struct {
	config   Config
	stmt     *clickhouse.SelectQuery
	cteNames map[string]bool // Tracks CTE names defined in WITH clause
}
