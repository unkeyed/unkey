package queryparser

import (
	"fmt"
	"slices"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (p *Parser) rewriteTables() error {
	if p.stmt.From == nil || p.stmt.From.Expr == nil {
		return fault.New("query must have FROM clause", fault.Public("Query must have a FROM clause"))
	}

	var rewriteErr error
	clickhouse.Walk(p.stmt.From.Expr, func(node clickhouse.Expr) bool {
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
			rewriteErr = fault.New(fmt.Sprintf("table '%s' not allowed", tableName), fault.Public(fmt.Sprintf("Access to table '%s' is not allowed", tableName)))
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
	// Block system tables
	if strings.HasPrefix(tableName, "system.") ||
		strings.HasPrefix(tableName, "information_schema.") ||
		strings.HasPrefix(tableName, "INFORMATION_SCHEMA.") {
		return false
	}

	if len(p.config.AllowedTables) == 0 {
		return true
	}

	return slices.Contains(p.config.AllowedTables, tableName)
}
