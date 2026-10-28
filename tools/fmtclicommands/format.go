package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"slices"
	"strconv"

	"golang.org/x/tools/imports"
)

func formatCommands(path string, source []byte) ([]byte, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, source, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	cliName := ""
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, err
		}
		if importPath == "github.com/unkeyed/unkey/pkg/cli" {
			cliName = "cli"
			if spec.Name != nil {
				cliName = spec.Name.Name
			}
		}
	}
	if cliName == "" {
		return source, nil
	}

	literals := make(map[*ast.CompositeLit]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		selector, ok := literal.Type.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Command" {
			return true
		}
		qualifier, ok := selector.X.(*ast.Ident)
		if ok && qualifier.Name == cliName {
			collectCommand(literal, literals)
		}
		return true
	})
	if len(literals) == 0 {
		return source, nil
	}

	tokenFile := fileSet.File(file.Pos())
	insertions := make(map[int]string)
	newlineBefore := func(pos token.Pos) {
		offset := tokenFile.Offset(pos)
		lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
		if len(bytes.TrimSpace(source[lineStart:offset])) > 0 {
			insertions[offset] += "\n"
		}
	}
	for literal := range literals {
		if len(literal.Elts) == 0 {
			continue
		}
		for _, element := range literal.Elts {
			newlineBefore(element.Pos())
		}
		end := tokenFile.Offset(literal.Elts[len(literal.Elts)-1].End())
		tail := source[end:tokenFile.Offset(literal.Rbrace)]
		var scan scanner.Scanner
		scan.Init(token.NewFileSet().AddFile("", -1, len(tail)), tail, nil, 0)
		if _, kind, _ := scan.Scan(); kind != token.COMMA {
			insertions[end] = "," + insertions[end]
		}
		newlineBefore(literal.Rbrace)
	}

	offsets := make([]int, 0, len(insertions))
	for offset := range insertions {
		offsets = append(offsets, offset)
	}
	slices.Sort(offsets)
	var expanded bytes.Buffer
	previous := 0
	for _, offset := range offsets {
		expanded.Write(source[previous:offset])
		expanded.WriteString(insertions[offset])
		previous = offset
	}
	expanded.Write(source[previous:])
	return imports.Process(path, expanded.Bytes(), &imports.Options{
		Fragment: false, AllErrors: true, Comments: true,
		TabIndent: true, TabWidth: 8, FormatOnly: true,
	})
}

func collectCommand(command *ast.CompositeLit, literals map[*ast.CompositeLit]bool) {
	if literals[command] {
		return
	}
	literals[command] = true
	for _, element := range command.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		name, ok := field.Key.(*ast.Ident)
		if !ok || name.Name != "Flags" && name.Name != "Examples" && name.Name != "Commands" {
			continue
		}
		list, ok := ast.Unparen(field.Value).(*ast.CompositeLit)
		if !ok {
			continue
		}
		literals[list] = true
		if name.Name == "Commands" {
			for _, element := range list.Elts {
				if nested, ok := ast.Unparen(element).(*ast.CompositeLit); ok {
					collectCommand(nested, literals)
				}
			}
		}
	}
}
