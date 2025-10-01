package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/xwb1989/sqlparser"
	"go.opentelemetry.io/otel/attribute"
)

// VirtualColumnValue represents a virtual column and its comparison value found in the query
type VirtualColumnValue struct {
	// VirtualColumn is the column name used in the query (e.g., "apiId")
	VirtualColumn string

	// Value is the value being compared for = comparisons (e.g., "api_123")
	Value string

	// Values is the list of values for IN comparisons (e.g., ["api_123", "api_456"])
	Values []string

	// ActualColumn is what it should be replaced with (e.g., "key_space_id")
	ActualColumn string

	// ActualValue is the looked-up value to use for = comparisons (e.g., the keyAuthId)
	// This is set by the caller after doing the lookup
	ActualValue string

	// ActualValues is the list of looked-up values for IN comparisons
	// This is set by the caller after doing the lookups
	ActualValues []string
}

// ExtractVirtualColumns extracts virtual column references from the query
// Returns a list of virtual columns that need to be resolved
func (r *Rewriter) ExtractVirtualColumns(userQuery string) ([]VirtualColumnValue, error) {
	if len(r.config.VirtualColumns) == 0 {
		return nil, nil
	}

	// Parse the SQL query
	stmt, err := sqlparser.Parse(userQuery)
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("invalid SQL query"))
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return nil, fault.New("only SELECT queries are allowed", fault.Public("Only SELECT queries are allowed"))
	}

	var virtualCols []VirtualColumnValue

	// Extract from WHERE clause
	if selectStmt.Where != nil {
		extracted := r.extractFromExpr(selectStmt.Where.Expr)
		virtualCols = append(virtualCols, extracted...)
	}

	// Extract from HAVING clause
	if selectStmt.Having != nil {
		extracted := r.extractFromExpr(selectStmt.Having.Expr)
		virtualCols = append(virtualCols, extracted...)
	}

	return virtualCols, nil
}

func (r *Rewriter) extractFromExpr(expr sqlparser.Expr) []VirtualColumnValue {
	var results []VirtualColumnValue

	switch node := expr.(type) {
	case *sqlparser.ComparisonExpr:
		// Check if left side is a virtual column
		if colName, ok := node.Left.(*sqlparser.ColName); ok {
			col := colName.Name.String()
			if virtualCol, isVirtual := r.config.VirtualColumns[col]; isVirtual {
				// Check if this is an IN comparison
				if node.Operator == sqlparser.InStr {
					// Extract multiple values from IN clause
					if values := r.extractValues(node.Right); len(values) > 0 {
						results = append(results, VirtualColumnValue{
							VirtualColumn: col,
							Values:        values,
							ActualColumn:  virtualCol.ActualColumn,
						})
					}
				} else {
					// Extract single value for = comparison
					if value := r.extractValue(node.Right); value != "" {
						results = append(results, VirtualColumnValue{
							VirtualColumn: col,
							Value:         value,
							ActualColumn:  virtualCol.ActualColumn,
						})
					}
				}
			}
		}

	case *sqlparser.AndExpr:
		results = append(results, r.extractFromExpr(node.Left)...)
		results = append(results, r.extractFromExpr(node.Right)...)

	case *sqlparser.OrExpr:
		results = append(results, r.extractFromExpr(node.Left)...)
		results = append(results, r.extractFromExpr(node.Right)...)

	case *sqlparser.ParenExpr:
		results = append(results, r.extractFromExpr(node.Expr)...)
	}

	return results
}

func (r *Rewriter) extractValue(expr sqlparser.Expr) string {
	switch node := expr.(type) {
	case *sqlparser.SQLVal:
		// Remove quotes from string values
		return strings.Trim(string(node.Val), "'\"")
	case sqlparser.ValTuple:
		// For IN clauses, use extractValues instead
		return ""
	}
	return ""
}

// extractValues extracts multiple values from an IN clause
func (r *Rewriter) extractValues(expr sqlparser.Expr) []string {
	switch node := expr.(type) {
	case sqlparser.ValTuple:
		var values []string
		for _, val := range node {
			if sqlVal, ok := val.(*sqlparser.SQLVal); ok {
				value := strings.Trim(string(sqlVal.Val), "'\"")
				if value != "" {
					values = append(values, value)
				}
			}
		}
		return values
	}
	return nil
}

// RewriteWithVirtualColumns rewrites the query replacing virtual columns with actual columns
// The virtualCols slice should have ActualValue populated by the caller
func (r *Rewriter) RewriteWithVirtualColumns(userQuery string, virtualCols []VirtualColumnValue) (string, error) {
	// Parse the SQL query
	stmt, err := sqlparser.Parse(userQuery)
	if err != nil {
		return "", fault.Wrap(err, fault.Public("invalid SQL query"))
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return "", fault.New("only SELECT queries are allowed", fault.Public("Only SELECT queries are allowed"))
	}

	// Build a lookup map
	virtualColMap := make(map[string]VirtualColumnValue)
	for _, vc := range virtualCols {
		var key string
		if len(vc.Values) > 0 {
			// IN clause - use Values
			key = fmt.Sprintf("%s=IN(%v)", vc.VirtualColumn, vc.Values)
		} else {
			// = comparison - use Value
			key = fmt.Sprintf("%s=%s", vc.VirtualColumn, vc.Value)
		}
		virtualColMap[key] = vc
	}

	// Rewrite WHERE clause
	if selectStmt.Where != nil {
		selectStmt.Where.Expr = r.rewriteExprVirtualCols(selectStmt.Where.Expr, virtualColMap)
	}

	// Rewrite HAVING clause
	if selectStmt.Having != nil {
		selectStmt.Having.Expr = r.rewriteExprVirtualCols(selectStmt.Having.Expr, virtualColMap)
	}

	// Now do the standard rewrite (workspace filter, table aliases, etc.)
	if err := r.rewriteSelect(selectStmt); err != nil {
		return "", err
	}

	// Convert back to SQL string
	rewritten := sqlparser.String(selectStmt)
	return rewritten, nil
}

func (r *Rewriter) rewriteExprVirtualCols(expr sqlparser.Expr, virtualColMap map[string]VirtualColumnValue) sqlparser.Expr {
	switch node := expr.(type) {
	case *sqlparser.ComparisonExpr:
		// Check if this is a virtual column comparison
		if colName, ok := node.Left.(*sqlparser.ColName); ok {
			col := colName.Name.String()

			// Handle IN clause
			if node.Operator == sqlparser.InStr {
				if values := r.extractValues(node.Right); len(values) > 0 {
					key := fmt.Sprintf("%s=IN(%v)", col, values)
					if vc, exists := virtualColMap[key]; exists && len(vc.ActualValues) > 0 {
						// Replace the column name
						node.Left = &sqlparser.ColName{
							Name: sqlparser.NewColIdent(vc.ActualColumn),
						}
						// Replace the values
						var newVals sqlparser.ValTuple
						for _, actualVal := range vc.ActualValues {
							newVals = append(newVals, sqlparser.NewStrVal([]byte(actualVal)))
						}
						node.Right = newVals
					}
				}
			} else {
				// Handle = comparison
				if value := r.extractValue(node.Right); value != "" {
					key := fmt.Sprintf("%s=%s", col, value)
					if vc, exists := virtualColMap[key]; exists {
						// Replace the column name
						node.Left = &sqlparser.ColName{
							Name: sqlparser.NewColIdent(vc.ActualColumn),
						}
						// Replace the value
						node.Right = sqlparser.NewStrVal([]byte(vc.ActualValue))
					}
				}
			}
		}
		return node

	case *sqlparser.AndExpr:
		node.Left = r.rewriteExprVirtualCols(node.Left, virtualColMap)
		node.Right = r.rewriteExprVirtualCols(node.Right, virtualColMap)
		return node

	case *sqlparser.OrExpr:
		node.Left = r.rewriteExprVirtualCols(node.Left, virtualColMap)
		node.Right = r.rewriteExprVirtualCols(node.Right, virtualColMap)
		return node

	case *sqlparser.ParenExpr:
		node.Expr = r.rewriteExprVirtualCols(node.Expr, virtualColMap)
		return node

	default:
		return expr
	}
}

// ResolveVirtualColumns resolves all virtual columns using the resolvers from the config.
// It collects unique IDs per virtual column, calls the appropriate resolver, and maps back to the virtual columns.
func (r *Rewriter) ResolveVirtualColumns(ctx context.Context, virtualCols []VirtualColumnValue) ([]VirtualColumnValue, error) {
	ctx, span := tracing.Start(ctx, "clickhouse.query.ResolveVirtualColumns")
	defer span.End()

	if len(virtualCols) == 0 {
		return virtualCols, nil
	}

	// Group IDs by virtual column type
	idsByColumn := make(map[string]map[string]bool)
	for _, vc := range virtualCols {
		if _, exists := idsByColumn[vc.VirtualColumn]; !exists {
			idsByColumn[vc.VirtualColumn] = make(map[string]bool)
		}

		if len(vc.Values) > 0 {
			// IN clause - collect all values
			for _, id := range vc.Values {
				idsByColumn[vc.VirtualColumn][id] = true
			}
		} else if vc.Value != "" {
			// = comparison - collect single value
			idsByColumn[vc.VirtualColumn][vc.Value] = true
		}
	}

	span.SetAttributes(attribute.Int("virtual_column_types", len(idsByColumn)))

	// Resolve each virtual column type
	lookupMaps := make(map[string]map[string]string)
	for virtualColName, ids := range idsByColumn {
		virtualCol, exists := r.config.VirtualColumns[virtualColName]
		if !exists {
			err := fault.New(fmt.Sprintf("no configuration for virtual column %s", virtualColName))
			tracing.RecordError(span, err)
			return nil, err
		}

		if virtualCol.Resolver == nil {
			err := fault.New(fmt.Sprintf("no resolver for virtual column %s", virtualColName))
			tracing.RecordError(span, err)
			return nil, err
		}

		// Convert map to slice
		idList := make([]string, 0, len(ids))
		for id := range ids {
			idList = append(idList, id)
		}

		// Call the resolver with tracing
		resolverCtx, resolverSpan := tracing.Start(ctx, fmt.Sprintf("clickhouse.query.ResolveVirtualColumn.%s", virtualColName))
		resolverSpan.SetAttributes(
			attribute.String("virtual_column", virtualColName),
			attribute.Int("id_count", len(idList)),
		)
		lookupMap, err := virtualCol.Resolver(resolverCtx, idList)
		resolverSpan.End()
		if err != nil {
			tracing.RecordError(span, err)
			return nil, fault.Wrap(err, fault.Public(fmt.Sprintf("Failed to resolve %s", virtualColName)))
		}

		lookupMaps[virtualColName] = lookupMap
	}

	// Map resolved values back to virtual columns
	resolved := make([]VirtualColumnValue, len(virtualCols))
	for i, vc := range virtualCols {
		resolved[i] = vc
		lookupMap := lookupMaps[vc.VirtualColumn]

		if len(vc.Values) > 0 {
			// IN clause - resolve multiple values
			for _, virtualID := range vc.Values {
				actualID, found := lookupMap[virtualID]
				if !found {
					err := fault.New(fmt.Sprintf("%s not found: %s", vc.VirtualColumn, virtualID))
					tracing.RecordError(span, err)
					return nil, err
				}
				resolved[i].ActualValues = append(resolved[i].ActualValues, actualID)
			}
		} else if vc.Value != "" {
			// = comparison - resolve single value
			actualID, found := lookupMap[vc.Value]
			if !found {
				err := fault.New(fmt.Sprintf("%s not found: %s", vc.VirtualColumn, vc.Value))
				tracing.RecordError(span, err)
				return nil, err
			}
			resolved[i].ActualValue = actualID
		}
	}

	return resolved, nil
}
