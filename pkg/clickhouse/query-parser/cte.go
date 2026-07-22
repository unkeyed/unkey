package queryparser

import (
	"fmt"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// buildCTERegistry scans the WITH clause and registers all CTE (Common Table Expression) names.
//
// CTEs are temporary named result sets defined with WITH clauses, like:
//
//	WITH cte_name AS (SELECT ...) SELECT * FROM cte_name
//
// We need to track CTE names separately because:
// 1. CTEs act as virtual tables that exist only for the duration of the query
// 2. When validating table names, we must distinguish between:
//   - Real tables (must be in allowedTables list)
//   - CTEs (allowed because they're user-defined subqueries)
//
// 3. Without this registry, we'd incorrectly reject valid queries that reference CTEs
//
// Example query that requires CTE tracking:
//
//	WITH filtered_data AS (SELECT * FROM key_verifications_v1 WHERE ...)
//	SELECT * FROM filtered_data
//
// Here, "filtered_data" is not a real table, but it's valid because it's a CTE.
func (p *Parser) buildCTERegistry() {
	clear(p.cteNames)
	if p.stmt.With == nil || len(p.stmt.With.CTEs) == 0 {
		return
	}

	for _, cte := range p.stmt.With.CTEs {
		// CTE Expr is the name we reference it by
		ident, ok := cte.Expr.(*clickhouse.Ident)
		if !ok {
			continue
		}

		p.cteNames[strings.ToLower(ident.Name)] = true
	}
}

// isCTE checks if a table name is a CTE
func (p *Parser) isCTE(tableName string) bool {
	return p.cteNames[strings.ToLower(tableName)]
}

// validateCTEAliases rejects ambiguous names before public aliases are
// rewritten to physical tables.
func (p *Parser) validateCTEAliases() error {
	publicAliases := make(map[string]string, len(p.config.TableAliases))
	for publicAlias := range p.config.TableAliases {
		publicAliases[strings.ToLower(publicAlias)] = publicAlias
	}

	var conflictingAlias string
	walkQueryIncludingExcept(p.stmt, func(node clickhouse.Expr) bool {
		query, ok := node.(*clickhouse.SelectQuery)
		if !ok || query.With == nil {
			return true
		}
		for _, cte := range query.With.CTEs {
			ident, isIdent := cte.Expr.(*clickhouse.Ident)
			if !isIdent {
				continue
			}
			if publicAlias, conflicts := publicAliases[strings.ToLower(ident.Name)]; conflicts {
				conflictingAlias = publicAlias
				return false
			}
		}
		return true
	})

	if conflictingAlias != "" {
		return fault.New(fmt.Sprintf("CTE name conflicts with public table alias '%s'", conflictingAlias),
			fault.Code(codes.User.BadRequest.InvalidAnalyticsTable.URN()),
			fault.Public(fmt.Sprintf("CTE name '%s' conflicts with a public table name", conflictingAlias)),
		)
	}
	return nil
}
