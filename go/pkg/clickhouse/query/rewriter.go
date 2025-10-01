package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/xwb1989/sqlparser"
)

// This package provides SQL query rewriting and validation for Clickhouse.
// It ensures queries are safe, workspace-isolated, and use proper table names.

// VirtualColumnResolver is a function that resolves virtual column values to actual values.
// It takes a list of virtual IDs and returns a map of virtual ID -> actual ID.
// The resolver should handle workspace validation and return an error if any ID is invalid.
type VirtualColumnResolver func(ctx context.Context, virtualIDs []string) (map[string]string, error)

// VirtualColumn represents a virtual column configuration
type VirtualColumn struct {
	// ActualColumn is the real column name in the database
	ActualColumn string

	// Resolver function that converts virtual IDs to actual IDs
	Resolver VirtualColumnResolver
}

// Config defines the query rewriting configuration
type Config struct {
	// WorkspaceID to inject into all queries
	WorkspaceID string

	// TableAliases maps user-friendly table names to actual Clickhouse table names
	// e.g., "key_verifications" -> "verifications.key_verifications_v1"
	TableAliases map[string]string

	// AllowedTables lists which tables can be queried (after alias resolution)
	AllowedTables []string

	// VirtualColumns maps virtual column names to their configuration
	// e.g., "apiId" -> VirtualColumn{ActualColumn: "key_space_id", Resolver: apiIdResolver}
	VirtualColumns map[string]VirtualColumn
}

// Rewriter rewrites and validates user SQL queries
type Rewriter struct {
	config Config
}

// New creates a new query rewriter
func New(config Config) *Rewriter {
	return &Rewriter{
		config: config,
	}
}

// Rewrite takes a user SQL query and:
// 1. Validates it's syntactically correct
// 2. Extracts and resolves virtual columns
// 3. Validates it's a safe SELECT query
// 4. Injects workspace_id filtering
// 5. Resolves table aliases
// 6. Validates table access
func (r *Rewriter) Rewrite(ctx context.Context, userQuery string) (string, error) {
	ctx, span := tracing.Start(ctx, "clickhouse.query.Rewrite")
	defer span.End()

	// Step 1: Validate syntax early (fail fast)
	_, span1 := tracing.Start(ctx, "clickhouse.query.ParseSyntax")
	stmt, err := sqlparser.Parse(userQuery)
	span1.End()
	if err != nil {
		tracing.RecordError(span, err)
		return "", fault.Wrap(err, fault.Public("Invalid SQL syntax"))
	}

	// Only allow SELECT queries
	if _, ok := stmt.(*sqlparser.Select); !ok {
		err := fault.New("only SELECT queries are allowed", fault.Public("Only SELECT queries are allowed"))
		tracing.RecordError(span, err)
		return "", err
	}

	// Step 2: Extract virtual columns from the query
	_, span2 := tracing.Start(ctx, "clickhouse.query.ExtractVirtualColumns")
	virtualCols, err := r.ExtractVirtualColumns(userQuery)
	span2.End()
	if err != nil {
		tracing.RecordError(span, err)
		return "", fault.Wrap(err, fault.Public("Invalid SQL query"))
	}

	// Step 3: Resolve virtual columns (batch database lookups)
	virtualCols, err = r.ResolveVirtualColumns(ctx, virtualCols)
	if err != nil {
		tracing.RecordError(span, err)
		return "", err
	}

	// Step 4: Rewrite query with resolved virtual columns and workspace filter
	_, span4 := tracing.Start(ctx, "clickhouse.query.RewriteWithVirtualColumns")
	safeQuery, err := r.RewriteWithVirtualColumns(userQuery, virtualCols)
	span4.End()
	if err != nil {
		tracing.RecordError(span, err)
		return "", fault.Wrap(err, fault.Public("Invalid SQL query"))
	}

	return safeQuery, nil
}

// rewriteQuery is the internal rewrite function that doesn't handle virtual columns
func (r *Rewriter) rewriteQuery(userQuery string) (string, error) {
	// Parse the SQL query
	stmt, err := sqlparser.Parse(userQuery)
	if err != nil {
		return "", fault.Wrap(err, fault.Public("invalid SQL query"))
	}

	// Only allow SELECT queries
	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return "", fault.New("only SELECT queries are allowed", fault.Public("Only SELECT queries are allowed"))
	}

	// Validate and rewrite the query
	if err := r.rewriteSelect(selectStmt); err != nil {
		return "", err
	}

	// Convert back to SQL string
	rewritten := sqlparser.String(selectStmt)
	return rewritten, nil
}

func (r *Rewriter) rewriteSelect(stmt *sqlparser.Select) error {
	// Check for subqueries and validate them recursively
	if err := r.validateExprs(stmt.SelectExprs); err != nil {
		return err
	}

	// Rewrite table names and validate access
	if err := r.rewriteTables(stmt); err != nil {
		return err
	}

	// Inject workspace_id filtering
	if err := r.injectWorkspaceFilter(stmt); err != nil {
		return err
	}

	// Validate WHERE clause (no dangerous functions)
	if stmt.Where != nil {
		if err := r.validateExpr(stmt.Where.Expr); err != nil {
			return err
		}
	}

	// Validate HAVING clause
	if stmt.Having != nil {
		if err := r.validateExpr(stmt.Having.Expr); err != nil {
			return err
		}
	}

	return nil
}

func (r *Rewriter) rewriteTables(stmt *sqlparser.Select) error {
	// Handle FROM clause
	if len(stmt.From) == 0 {
		return fault.New("query must have a FROM clause", fault.Public("Query must have a FROM clause"))
	}

	for _, tableExpr := range stmt.From {
		if err := r.rewriteTableExpr(tableExpr); err != nil {
			return err
		}
	}

	return nil
}

func (r *Rewriter) rewriteTableExpr(expr sqlparser.TableExpr) error {
	switch node := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		// Get the table name
		tableName := sqlparser.String(node.Expr)
		tableName = strings.Trim(tableName, "`\"")

		// Check if it's an alias and resolve it
		if actualTable, ok := r.config.TableAliases[tableName]; ok {
			tableName = actualTable
		}

		// Validate table access
		if !r.isTableAllowed(tableName) {
			msg := fmt.Sprintf("access to table '%s' is not allowed", tableName)
			return fault.New(msg, fault.Public(msg))
		}

		// Update the table name in the AST
		// Split database.table format if present
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			node.Expr = sqlparser.TableName{
				Qualifier: sqlparser.NewTableIdent(parts[0]),
				Name:      sqlparser.NewTableIdent(parts[1]),
			}
		} else {
			node.Expr = sqlparser.TableName{
				Name: sqlparser.NewTableIdent(tableName),
			}
		}

	case *sqlparser.JoinTableExpr:
		// Recursively handle joins
		if err := r.rewriteTableExpr(node.LeftExpr); err != nil {
			return err
		}
		if err := r.rewriteTableExpr(node.RightExpr); err != nil {
			return err
		}

	case *sqlparser.ParenTableExpr:
		// Handle parenthesized table expressions
		for _, te := range node.Exprs {
			if err := r.rewriteTableExpr(te); err != nil {
				return err
			}
		}

	default:
		msg := fmt.Sprintf("unsupported table expression type: %T", expr)
		return fault.New(msg, fault.Public("Unsupported query structure"))
	}

	return nil
}

func (r *Rewriter) isTableAllowed(tableName string) bool {
	// Block system tables
	if strings.HasPrefix(tableName, "system.") ||
		strings.HasPrefix(tableName, "information_schema.") ||
		strings.HasPrefix(tableName, "INFORMATION_SCHEMA.") {
		return false
	}

	// If no allowed tables configured, allow all non-system tables
	if len(r.config.AllowedTables) == 0 {
		return true
	}

	// Check if table is in allowed list
	for _, allowed := range r.config.AllowedTables {
		if tableName == allowed {
			return true
		}
	}

	return false
}

func (r *Rewriter) injectWorkspaceFilter(stmt *sqlparser.Select) error {
	// Create workspace_id filter: workspace_id = 'ws_xxx'
	workspaceFilter := &sqlparser.ComparisonExpr{
		Operator: sqlparser.EqualStr,
		Left: &sqlparser.ColName{
			Name: sqlparser.NewColIdent("workspace_id"),
		},
		Right: sqlparser.NewStrVal([]byte(r.config.WorkspaceID)),
	}

	// Inject into WHERE clause
	if stmt.Where == nil {
		// No WHERE clause exists, create one
		stmt.Where = &sqlparser.Where{
			Type: sqlparser.WhereStr,
			Expr: workspaceFilter,
		}
	} else {
		// Combine with existing WHERE using AND
		stmt.Where.Expr = &sqlparser.AndExpr{
			Left:  workspaceFilter,
			Right: stmt.Where.Expr,
		}
	}

	return nil
}

func (r *Rewriter) validateExprs(exprs sqlparser.SelectExprs) error {
	for _, expr := range exprs {
		switch node := expr.(type) {
		case *sqlparser.StarExpr:
			// SELECT * is allowed
			continue
		case *sqlparser.AliasedExpr:
			if err := r.validateExpr(node.Expr); err != nil {
				return err
			}
		default:
			msg := fmt.Sprintf("unsupported select expression type: %T", expr)
			return fault.New(msg, fault.Public("Unsupported query structure"))
		}
	}
	return nil
}

func (r *Rewriter) validateExpr(expr sqlparser.Expr) error {
	switch node := expr.(type) {
	case *sqlparser.ComparisonExpr:
		if err := r.validateExpr(node.Left); err != nil {
			return err
		}
		if err := r.validateExpr(node.Right); err != nil {
			return err
		}

	case *sqlparser.AndExpr:
		if err := r.validateExpr(node.Left); err != nil {
			return err
		}
		if err := r.validateExpr(node.Right); err != nil {
			return err
		}

	case *sqlparser.OrExpr:
		if err := r.validateExpr(node.Left); err != nil {
			return err
		}
		if err := r.validateExpr(node.Right); err != nil {
			return err
		}

	case *sqlparser.FuncExpr:
		// Check for dangerous functions
		funcName := strings.ToLower(node.Name.String())
		if r.isDangerousFunction(funcName) {
			msg := fmt.Sprintf("function '%s' is not allowed", funcName)
			return fault.New(msg, fault.Public(msg))
		}

		// Validate function arguments
		for _, arg := range node.Exprs {
			if aliasedExpr, ok := arg.(*sqlparser.AliasedExpr); ok {
				if err := r.validateExpr(aliasedExpr.Expr); err != nil {
					return err
				}
			}
		}

	case *sqlparser.Subquery:
		// Validate subqueries
		if selectStmt, ok := node.Select.(*sqlparser.Select); ok {
			if err := r.rewriteSelect(selectStmt); err != nil {
				return err
			}
		} else {
			return fault.New("only SELECT subqueries are allowed", fault.Public("Only SELECT subqueries are allowed"))
		}

	case *sqlparser.ColName, *sqlparser.SQLVal, sqlparser.BoolVal,
		*sqlparser.NullVal, sqlparser.ValTuple:
		// These are safe basic types
		return nil

	default:
		// Allow other common expression types
		return nil
	}

	return nil
}

func (r *Rewriter) isDangerousFunction(funcName string) bool {
	// Block functions that can access files, execute commands, or modify data
	dangerousFunctions := []string{
		"file",
		"input",
		"output",
		"url",
		"executable",
		"system",
		"shell",
		"pipe",
	}

	for _, dangerous := range dangerousFunctions {
		if funcName == dangerous {
			return true
		}
	}

	return false
}
