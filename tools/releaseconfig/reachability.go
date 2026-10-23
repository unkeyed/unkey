package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type reachability struct {
	packages map[string]goPackage
	files    map[string]bool
	symbols  map[string]bool
}

func (analysis *reachability) link(entrypoint string, tags, env []string) (err error) {
	dir, err := os.MkdirTemp("", "release-reachability-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	cmd := exec.Command("go", "build", "-mod=readonly", "-buildvcs=false", "-gcflags=./...=-l", "-ldflags=-dumpdep", "-tags="+strings.Join(tags, ","), "-o", filepath.Join(dir, "release"), entrypoint)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("link %s: %s: %w", entrypoint, output, err)
	}
	edges := 0
	for line := range strings.SplitSeq(string(output), "\n") {
		_, symbol, ok := strings.Cut(line, " -> ")
		if !ok {
			continue
		}
		edges++
		symbol, _, _ = strings.Cut(symbol, " <")
		analysis.symbols[withoutTypeArguments(symbol)] = true
	}
	if edges == 0 {
		return errors.New("linker emitted no reachability graph")
	}
	return nil
}

func withoutTypeArguments(name string) string {
	var result strings.Builder
	depth := 0
	for _, character := range name {
		switch character {
		case '[':
			depth++
		case ']':
			depth--
		default:
			if depth == 0 {
				result.WriteRune(character)
			}
		}
	}
	return result.String()
}

type declaration struct {
	text    string
	symbols []string
	reason  string
}

func declarations(source []byte, pkg goPackage) (map[string]declaration, error) {
	result := map[string]declaration{}
	if len(source) == 0 {
		return result, nil
	}
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "", source, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	prefix := pkg.ImportPath
	if pkg.Name == "main" {
		prefix = "main"
	}
	prefix += "."
	text := func(node ast.Node) string {
		start := positions.Position(node.Pos()).Offset
		end := positions.Position(node.End()).Offset
		var lexer scanner.Scanner
		lexer.Init(token.NewFileSet().AddFile("", -1, end-start), source[start:end], nil, 0)
		var tokens strings.Builder
		for {
			_, kind, literal := lexer.Scan()
			if kind == token.EOF {
				return tokens.String()
			}
			fmt.Fprintf(&tokens, "%d:%s\x00", kind, literal)
		}
	}
	foreignSymbols := false
	for _, group := range file.Comments {
		for _, comment := range group.List {
			foreignSymbols = foreignSymbols || strings.HasPrefix(comment.Text, "//go:linkname") || strings.HasPrefix(comment.Text, "//export")
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build") {
				entry := result["#directives"]
				entry.text += comment.Text + "\n"
				entry.reason = "compiler directive (conservative)"
				result["#directives"] = entry
			}
		}
	}
	for _, node := range file.Decls {
		switch node := node.(type) {
		case *ast.FuncDecl:
			name := node.Name.Name
			symbols := []string{prefix + name}
			if node.Recv != nil {
				receiver := withoutTypeArguments(textReceiver(node.Recv.List[0].Type))
				name = receiver + "." + name
				symbols = []string{prefix + name, prefix + "(*" + receiver + ")." + node.Name.Name}
			}
			entry := declaration{text: text(node), symbols: symbols, reason: ""}
			if foreignSymbols {
				entry.reason = "foreign symbol names (conservative)"
			}
			if name == "init" {
				entry.text = result[name].text + entry.text
				entry.reason = "package initialization (conservative)"
			}
			result[name] = entry
		case *ast.GenDecl:
			if node.Tok == token.VAR {
				entry := result["#variables"]
				entry.text += text(node)
				entry.reason = "variable initialization or order (conservative)"
				result["#variables"] = entry
			}
			if node.Tok == token.IMPORT {
				for _, spec := range node.Specs {
					entry := result["#imports"]
					entry.text += text(spec)
					entry.reason = "imports or initialization order (conservative)"
					result["#imports"] = entry
				}
				continue
			}
			for _, spec := range node.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					value := text(spec)
					if node.Tok == token.CONST {
						value = text(node)
					}
					for _, name := range spec.Names {
						result[name.Name] = declaration{
							text: value, symbols: []string{prefix + name.Name},
							reason: node.Tok.String() + " " + name.Name + " (conservative)",
						}
					}
				case *ast.TypeSpec:
					result[spec.Name.Name] = declaration{text: text(spec), symbols: nil, reason: "type " + spec.Name.Name + " (conservative)"}
				}
			}
		}
	}
	return result, nil
}

func textReceiver(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.StarExpr:
		return textReceiver(expression.X)
	case *ast.IndexExpr:
		return textReceiver(expression.X)
	case *ast.IndexListExpr:
		return textReceiver(expression.X)
	case *ast.Ident:
		return expression.Name
	default:
		return ""
	}
}

func (analysis *reachability) changedGo(sourceBefore, sourceAfter []byte, pkg goPackage) string {
	before, err := declarations(sourceBefore, pkg)
	if err != nil {
		return "historical Go syntax (conservative)"
	}
	after, err := declarations(sourceAfter, pkg)
	if err != nil {
		return "historical Go syntax (conservative)"
	}
	names := make([]string, 0, len(after)+len(before))
	for name := range after {
		names = append(names, name)
	}
	for name := range before {
		names = append(names, name)
	}
	slices.Sort(names)
	fallback := ""
	for _, name := range slices.Compact(names) {
		old, next := before[name], after[name]
		if old.text == next.text {
			continue
		}
		for _, symbol := range append(old.symbols, next.symbols...) {
			if analysis.symbols[symbol] {
				return "linked `" + symbol + "`"
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
	return fallback
}

func (analysis *reachability) embedded(dir string, pkg goPackage) (bool, error) {
	for filename := range analysis.files {
		if filepath.ToSlash(filepath.Dir(filename)) != dir {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ParseComments)
		if err != nil {
			return false, err
		}
		for _, node := range file.Decls {
			group, ok := node.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR {
				continue
			}
			for _, node := range group.Specs {
				spec, ok := node.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, doc := range []*ast.CommentGroup{group.Doc, spec.Doc} {
					if doc == nil {
						continue
					}
					for _, comment := range doc.List {
						if !strings.HasPrefix(comment.Text, "//go:embed") {
							continue
						}
						prefix := pkg.ImportPath
						if pkg.Name == "main" {
							prefix = "main"
						}
						for _, name := range spec.Names {
							if analysis.symbols[prefix+"."+name.Name] {
								return true, nil
							}
						}
					}
				}
			}
		}
	}
	return false, nil
}

func gitOutput(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s: %w", strings.Join(args, " "), output, err)
	}
	return string(output), nil
}
