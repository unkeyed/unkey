package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"maps"
	"slices"
	"strings"
)

type declaration struct {
	text    string
	symbols []string
	reason  string
}

type declarations struct {
	named      map[string]declaration
	directives string
	imports    string
	variables  string
}

type sourceFile struct {
	contents  []byte
	positions *token.FileSet
}

func (analysis *reachability) changedGo(sourceBefore, sourceAfter []byte, pkg goPackage) changeImpact {
	before, err := parseDeclarations(sourceBefore, pkg)
	if err != nil {
		return changeImpact{symbol: "", reason: "historical Go syntax (conservative)"}
	}
	after, err := parseDeclarations(sourceAfter, pkg)
	if err != nil {
		return changeImpact{symbol: "", reason: "historical Go syntax (conservative)"}
	}

	fallback := ""
	if before.directives != after.directives {
		fallback = "compiler directive (conservative)"
	}
	if before.imports != after.imports {
		fallback = "imports or initialization order (conservative)"
	}
	if before.variables != after.variables {
		fallback = "variable initialization or order (conservative)"
	}
	names := slices.AppendSeq(slices.Collect(maps.Keys(before.named)), maps.Keys(after.named))
	slices.Sort(names)
	for _, name := range slices.Compact(names) {
		old, next := before.named[name], after.named[name]
		if old.text == next.text {
			continue
		}
		for _, symbol := range slices.Concat(old.symbols, next.symbols) {
			if analysis.symbols[symbol] {
				return changeImpact{symbol: symbol, reason: ""}
			}
		}
		switch {
		case next.reason != "":
			fallback = next.reason
		case old.reason != "":
			fallback = old.reason
		case next.text == "":
			fallback = "removed Go declaration (conservative)"
		}
	}
	return changeImpact{symbol: "", reason: fallback}
}

func parseDeclarations(contents []byte, pkg goPackage) (declarations, error) {
	result := declarations{named: map[string]declaration{}, directives: "", imports: "", variables: ""}
	if len(contents) == 0 {
		return result, nil
	}
	source := sourceFile{contents: contents, positions: token.NewFileSet()}
	file, err := parser.ParseFile(source.positions, "", contents, parser.ParseComments)
	if err != nil {
		return result, err
	}

	foreignSymbols := false
	for _, group := range file.Comments {
		for _, comment := range group.List {
			foreignSymbols = foreignSymbols || strings.HasPrefix(comment.Text, "//go:linkname") || strings.HasPrefix(comment.Text, "//export")
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build") {
				result.directives += comment.Text + "\n"
			}
		}
	}
	for _, node := range file.Decls {
		switch node := node.(type) {
		case *ast.FuncDecl:
			name, entry := source.function(node, pkg.symbolPrefix())
			if foreignSymbols {
				entry.reason = "foreign symbol names (conservative)"
			}
			if name == "init" {
				entry.text = result.named[name].text + entry.text
				entry.reason = "package initialization (conservative)"
			}
			result.named[name] = entry
		case *ast.GenDecl:
			source.addGroup(&result, node, pkg.symbolPrefix())
		}
	}
	return result, nil
}

func (source sourceFile) function(node *ast.FuncDecl, prefix string) (string, declaration) {
	name := node.Name.Name
	symbols := []string{prefix + name}
	if node.Recv != nil {
		receiver := receiverName(node.Recv.List[0].Type)
		name = receiver + "." + name
		symbols = []string{prefix + name, prefix + "(*" + receiver + ")." + node.Name.Name}
	}
	return name, declaration{text: source.text(node), symbols: symbols, reason: ""}
}

func (source sourceFile) addGroup(result *declarations, group *ast.GenDecl, prefix string) {
	if group.Tok == token.VAR {
		result.variables += source.text(group)
	}
	for _, spec := range group.Specs {
		switch spec := spec.(type) {
		case *ast.ImportSpec:
			result.imports += source.text(spec)
		case *ast.ValueSpec:
			value := source.text(spec)
			if group.Tok == token.CONST {
				value = source.text(group)
			}
			for _, name := range spec.Names {
				result.named[name.Name] = declaration{
					text:    value,
					symbols: []string{prefix + name.Name},
					reason:  group.Tok.String() + " " + name.Name + " (conservative)",
				}
			}
		case *ast.TypeSpec:
			result.named[spec.Name.Name] = declaration{
				text: source.text(spec), symbols: nil, reason: "type " + spec.Name.Name + " (conservative)",
			}
		}
	}
}

func (source sourceFile) text(node ast.Node) string {
	start := source.positions.Position(node.Pos()).Offset
	end := source.positions.Position(node.End()).Offset
	var lexer scanner.Scanner
	lexer.Init(token.NewFileSet().AddFile("", -1, end-start), source.contents[start:end], nil, 0)
	var tokens strings.Builder
	for {
		_, kind, literal := lexer.Scan()
		if kind == token.EOF {
			return tokens.String()
		}
		fmt.Fprintf(&tokens, "%d:%s\x00", kind, literal)
	}
}

func receiverName(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.StarExpr:
		return receiverName(expression.X)
	case *ast.IndexExpr:
		return receiverName(expression.X)
	case *ast.IndexListExpr:
		return receiverName(expression.X)
	case *ast.Ident:
		return expression.Name
	default:
		return ""
	}
}
