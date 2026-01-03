package queryparser

import (
	"fmt"
	"slices"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

func (p *Parser) rewriteTables() error {
	if p.stmt.From == nil || p.stmt.From.Expr == nil {
		return fault.New("query must have FROM clause",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("Query must have a FROM clause"),
		)
	}

	var rewriteErr error

	// Walk the ENTIRE statement to find all tables, including those in UNION queries
	clickhouse.WalkWithBreak(p.stmt, func(node clickhouse.Expr) bool {
		tableIdent, ok := node.(*clickhouse.TableIdentifier)
		if !ok {
			return true
		}

		// Get table name
		tableName := tableIdent.Table.Name
		if tableIdent.Database != nil {
			tableName = tableIdent.Database.Name + "." + tableIdent.Table.Name
		}

		// Resolve alias
		if actualTable, ok := p.config.TableAliases[tableName]; ok {
			tableName = actualTable
		}

		// Validate access
		if !p.isTableAllowed(tableName) {
			rewriteErr = fault.New(fmt.Sprintf("table '%s' not allowed", tableName),
				fault.Code(codes.User.BadRequest.InvalidAnalyticsTable.URN()),
				fault.Public(fmt.Sprintf("Access to table '%s' is not allowed", tableName)),
			)

			return false
		}

		// Update AST
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			tableIdent.Database = &clickhouse.Ident{Name: parts[0]}
			tableIdent.Table = &clickhouse.Ident{Name: parts[1]}
		} else {
			tableIdent.Database = nil
			tableIdent.Table = &clickhouse.Ident{Name: tableName}
		}

		return true
	})

	return rewriteErr
}

func (p *Parser) isTableAllowed(tableName string) bool {
	// Check if it's a CTE first - CTEs are always allowed, as we check if the cbe is using allowed tables as well
	if p.isCTE(tableName) {
		return true
	}

	if len(p.config.AllowedTables) == 0 {
		return false
	}

	// Always block system and information_schema tables
	lowerTableName := strings.ToLower(tableName)
	if strings.HasPrefix(lowerTableName, "system.") || strings.HasPrefix(lowerTableName, "information_schema.") {
		return false
	}

	return slices.Contains(p.config.AllowedTables, tableName)
}
