package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

func (p *Parser) buildAliasMap() {
	p.aliasMap = make(map[string]string)

	for canonicalName, vcConfig := range p.config.VirtualColumns {
		// Map canonical name to itself
		p.aliasMap[canonicalName] = canonicalName

		// Map all aliases to canonical name
		for _, alias := range vcConfig.Aliases {
			p.aliasMap[alias] = canonicalName
		}
	}
}

func (p *Parser) resolveVirtualColumnName(name string) (string, bool) {
	canonical, ok := p.aliasMap[name]
	return canonical, ok
}

func (p *Parser) extractValues(expr clickhouse.Expr, canonicalName string, virtualCols map[string][]string) {
	// Handle single string literal
	if strLit, ok := expr.(*clickhouse.StringLiteral); ok {
		virtualCols[canonicalName] = append(virtualCols[canonicalName], strLit.Literal)
		return
	}

	// Handle IN operator: IN ('val1', 'val2', 'val3')
	if paramList, ok := expr.(*clickhouse.ParamExprList); ok {
		if paramList.Items != nil {
			for _, item := range paramList.Items.Items {
				// Each item is wrapped in a ColumnExpr
				if colExpr, ok := item.(*clickhouse.ColumnExpr); ok {
					if strLit, ok := colExpr.Expr.(*clickhouse.StringLiteral); ok {
						virtualCols[canonicalName] = append(virtualCols[canonicalName], strLit.Literal)
					}
				}
			}
		}
	}
}

func (p *Parser) rewriteValues(expr clickhouse.Expr, resolvedMap map[string]string) {
	// Handle single string literal
	if strLit, ok := expr.(*clickhouse.StringLiteral); ok {
		if resolved, ok := resolvedMap[strLit.Literal]; ok {
			strLit.Literal = resolved
		}
		return
	}

	// Handle IN operator: IN ('val1', 'val2', 'val3')
	if paramList, ok := expr.(*clickhouse.ParamExprList); ok {
		if paramList.Items != nil {
			for _, item := range paramList.Items.Items {
				// Each item is wrapped in a ColumnExpr
				if colExpr, ok := item.(*clickhouse.ColumnExpr); ok {
					if strLit, ok := colExpr.Expr.(*clickhouse.StringLiteral); ok {
						if resolved, ok := resolvedMap[strLit.Literal]; ok {
							strLit.Literal = resolved
						}
					}
				}
			}
		}
	}
}

func (p *Parser) rewriteVirtualColumns(ctx context.Context) error {
	if len(p.config.VirtualColumns) == 0 {
		return nil
	}

	// Extract virtual columns from WHERE and HAVING
	virtualCols := make(map[string][]string) // virtualCol -> list of IDs
	extractFunc := func(node clickhouse.Expr) bool {
		binOp, ok := node.(*clickhouse.BinaryOperation)
		if !ok {
			return true
		}

		// Check if left is a column name we care about
		colName := ""
		if ident, ok := binOp.LeftExpr.(*clickhouse.Ident); ok {
			colName = ident.Name
		} else if nestedIdent, ok := binOp.LeftExpr.(*clickhouse.NestedIdentifier); ok {
			if nestedIdent.Ident != nil {
				colName = nestedIdent.Ident.Name
			}
		}

		if colName == "" {
			return true
		}

		// Resolve alias to canonical name
		canonicalName, isVirtual := p.resolveVirtualColumnName(colName)
		if !isVirtual {
			return true
		}

		// Extract values from right side (handles =, !=, <, >, IN, etc.)
		p.extractValues(binOp.RightExpr, canonicalName, virtualCols)
		return true
	}

	if p.stmt.Where != nil {
		clickhouse.Walk(p.stmt.Where.Expr, extractFunc)
	}

	if p.stmt.Having != nil {
		clickhouse.Walk(p.stmt.Having.Expr, extractFunc)
	}

	// Resolve each virtual column
	resolvedMaps := make(map[string]map[string]string)
	for virtualCol, ids := range virtualCols {
		vcConfig := p.config.VirtualColumns[virtualCol]
		if vcConfig.Resolver == nil {
			return fault.New(fmt.Sprintf("no resolver for %s", virtualCol))
		}

		resolved, err := vcConfig.Resolver(ctx, ids)
		if err != nil {
			return fault.Wrap(err, fault.Public(fmt.Sprintf("Failed to resolve %s", virtualCol)))
		}

		resolvedMaps[virtualCol] = resolved
	}

	// Rewrite the AST by walking and modifying in place
	rewriteFunc := func(node clickhouse.Expr) bool {
		binOp, ok := node.(*clickhouse.BinaryOperation)
		if !ok {
			return true
		}

		colName := ""
		var identToModify *clickhouse.Ident

		if ident, ok := binOp.LeftExpr.(*clickhouse.Ident); ok {
			colName = ident.Name
			identToModify = ident
		} else if nestedIdent, ok := binOp.LeftExpr.(*clickhouse.NestedIdentifier); ok {
			if nestedIdent.Ident != nil {
				colName = nestedIdent.Ident.Name
				identToModify = nestedIdent.Ident
			}
		}

		if colName == "" || identToModify == nil {
			return true
		}

		// Resolve alias to canonical name
		canonicalName, isVirtual := p.resolveVirtualColumnName(colName)
		if !isVirtual {
			return true
		}

		vcConfig := p.config.VirtualColumns[canonicalName]
		// Replace column name with actual column
		identToModify.Name = vcConfig.ActualColumn

		// Replace values in right expression
		p.rewriteValues(binOp.RightExpr, resolvedMaps[canonicalName])
		return true
	}

	if p.stmt.Where != nil {
		clickhouse.Walk(p.stmt.Where.Expr, rewriteFunc)
	}
	if p.stmt.Having != nil {
		clickhouse.Walk(p.stmt.Having.Expr, rewriteFunc)
	}

	return nil
}
