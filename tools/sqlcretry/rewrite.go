package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

const mysqlImport = "github.com/unkeyed/unkey/pkg/mysql"

type edit struct {
	start int
	end   int
	text  string
}

func rewrite(source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	queries := map[string]bool{}
	for _, declaration := range file.Decls {
		declaration, ok := declaration.(*ast.GenDecl)
		if !ok || declaration.Tok != token.CONST {
			continue
		}
		for _, spec := range declaration.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Values) != 1 {
				continue
			}
			expression := value.Values[0]
			literal, err := types.Eval(fset, nil, token.NoPos, string(source[fset.Position(expression.Pos()).Offset:fset.Position(expression.End()).Offset]))
			if err != nil || literal.Value == nil || literal.Value.Kind() != constant.String {
				continue
			}
			query := constant.StringVal(literal.Value)
			header, _, _ := strings.Cut(query, "\n")
			fields := strings.Fields(header)
			if len(fields) != 4 || fields[0] != "--" || fields[1] != "name:" || (fields[3] != ":one" && fields[3] != ":many") {
				continue
			}
			safe, err := retrySafe(query)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", fields[2], err)
			}
			if safe {
				queries[fields[2]] = true
			}
		}
	}

	var edits []edit
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || method.Body == nil {
			continue
		}
		if !queries[method.Name.Name] {
			continue
		}
		delete(queries, method.Name.Name)
		if wrapped(method) {
			continue
		}
		wrapper, err := wrapperSource(fset, source, method)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", method.Name.Name, err)
		}
		start := fset.Position(method.Pos()).Offset
		nameStart := fset.Position(method.Name.Pos()).Offset
		nameEnd := fset.Position(method.Name.End()).Offset
		edits = append(edits, edit{
			start: start,
			end:   nameEnd,
			text:  wrapper + "\n\n" + string(source[start:nameStart]) + strings.ToLower(method.Name.Name[:1]) + method.Name.Name[1:],
		})
	}
	for query := range queries {
		return nil, fmt.Errorf("%s: generated query method not found", query)
	}
	if len(edits) == 0 {
		return source, nil
	}
	slices.Reverse(edits)
	for _, edit := range edits {
		source = slices.Concat(source[:edit.start], []byte(edit.text), source[edit.end:])
	}

	fset = token.NewFileSet()
	file, err = parser.ParseFile(fset, "", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	astutil.AddImport(fset, file, mysqlImport)
	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, err
	}
	return imports.Process("", output.Bytes(), nil)
}

func wrapped(method *ast.FuncDecl) bool {
	if len(method.Body.List) != 1 {
		return false
	}
	statement, ok := method.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return false
	}
	call, ok := statement.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "ReadWithRetry" {
		return false
	}
	name, ok := selector.X.(*ast.Ident)
	return ok && name.Name == "mysql"
}

func wrapperSource(fset *token.FileSet, source []byte, method *ast.FuncDecl) (string, error) {
	if len(method.Recv.List) != 1 || len(method.Recv.List[0].Names) != 1 || method.Type.Results == nil || len(method.Type.Results.List) != 2 {
		return "", fmt.Errorf("unsupported sqlc method signature")
	}
	pointer, ok := method.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", fmt.Errorf("expected a *Queries receiver")
	}
	receiverType, ok := pointer.X.(*ast.Ident)
	if !ok || receiverType.Name != "Queries" {
		return "", fmt.Errorf("expected a *Queries receiver")
	}
	receiver := method.Recv.List[0].Names[0].Name
	database := receiver + ".db"
	var arguments []string
	for _, field := range method.Type.Params.List {
		for _, name := range field.Names {
			arguments = append(arguments, name.Name)
			if fieldType, ok := field.Type.(*ast.Ident); ok && fieldType.Name == "DBTX" {
				database = name.Name
			}
		}
	}
	if len(arguments) == 0 || arguments[0] != "ctx" {
		return "", fmt.Errorf("expected context as the first sqlc parameter")
	}
	for _, field := range method.Type.Params.List {
		if _, variadic := field.Type.(*ast.Ellipsis); variadic {
			arguments[len(arguments)-1] += "..."
		}
	}
	header := string(source[fset.Position(method.Pos()).Offset:fset.Position(method.Body.Lbrace).Offset])
	results := string(source[fset.Position(method.Type.Results.Pos()).Offset:fset.Position(method.Type.Results.End()).Offset])
	raw := strings.ToLower(method.Name.Name[:1]) + method.Name.Name[1:]
	return fmt.Sprintf("%s{\n\treturn mysql.ReadWithRetry(ctx, %s, func() %s {\n\t\treturn %s.%s(%s)\n\t})\n}", header, database, results, receiver, raw, strings.Join(arguments, ", ")), nil
}
