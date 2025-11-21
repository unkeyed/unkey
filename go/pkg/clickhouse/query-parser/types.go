package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// SecurityFilter represents a row-level security constraint
type SecurityFilter struct {
	Column        string   // Column name
	AllowedValues []string // Values user is allowed to access
}

// Config for the parser
type Config struct {
	WorkspaceID       string
	TableAliases      map[string]string
	AllowedTables     []string
	SecurityFilters   []SecurityFilter // Row-level security filters (auto-injected)
	Limit             int
	MaxQueryRangeDays int32 // Maximum historical data range user can query in days
	Logger            logging.Logger
}

// Parser rewrites ClickHouse queries
type Parser struct {
	config   Config
	logger   logging.Logger
	stmt     *clickhouse.SelectQuery
	cteNames map[string]bool // Tracks CTE names defined in WITH clause
}
