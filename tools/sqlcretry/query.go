package main

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
)

func retrySafe(query string) (bool, error) {
	if strings.Contains(query, "/*!") || strings.Contains(query, "/*T!") {
		return false, nil
	}
	statement, err := parser.New().ParseOneStmt(query, "", "")
	if err != nil {
		return false, fmt.Errorf("parse query for retry safety: %w", err)
	}
	switch statement.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt:
	default:
		return false, nil
	}

	visitor := &readSafety{safe: true}
	_, safe := statement.Accept(visitor)
	return safe, nil
}

type readSafety struct {
	safe bool
}

func (v *readSafety) Enter(node ast.Node) (ast.Node, bool) {
	switch node := node.(type) {
	case *ast.SelectStmt:
		v.safe = (node.LockInfo == nil || node.LockInfo.LockType == ast.SelectLockNone) && node.SelectIntoOpt == nil && !node.CalcFoundRows
	case *ast.VariableExpr, *ast.Assignment, *ast.VariableAssignment, *ast.TableOptimizerHint:
		v.safe = false
	case *ast.FuncCallExpr:
		if node.Schema.O != "" {
			v.safe = false
			break
		}
		switch node.FnName.L {
		case "coalesce", "current_timestamp", "date_add", "date_sub", "floor", "greatest", "json_array", "json_object", "json_value", "lower", "nullif", "now", "unix_timestamp":
		default:
			v.safe = false
		}
	}
	return node, !v.safe
}

func (v *readSafety) Leave(node ast.Node) (ast.Node, bool) {
	return node, v.safe
}
