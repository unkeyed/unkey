package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type assetUsage struct {
	embedded bool
	linked   bool
	possible bool
}

func (analysis *reachability) assetUsage(filename string) (assetUsage, error) {
	var usage assetUsage
	for dir, pkg := range analysis.packages {
		relative, err := filepath.Rel(dir, filename)
		if err != nil {
			return usage, err
		}
		relative = filepath.ToSlash(relative)
		matched, err := matchesEmbedPattern(relative, pkg.EmbedPatterns)
		if err != nil {
			return usage, err
		}
		usage.possible = usage.possible || matched
		if !slices.Contains(pkg.EmbedFiles, relative) {
			continue
		}
		usage.embedded = true
		usage.linked, err = analysis.packageEmbedsLinked(dir, pkg)
		if err != nil || usage.linked {
			return usage, err
		}
	}
	return usage, nil
}

func matchesEmbedPattern(filename string, patterns []string) (bool, error) {
	matched := false
	for _, pattern := range patterns {
		for candidate := filename; candidate != "." && candidate != ".."; candidate = path.Dir(candidate) {
			match, err := path.Match(strings.TrimPrefix(pattern, "all:"), candidate)
			if err != nil {
				return false, err
			}
			matched = matched || match
		}
	}
	return matched, nil
}

func (analysis *reachability) packageEmbedsLinked(dir string, pkg goPackage) (bool, error) {
	for filename := range analysis.files {
		if filepath.ToSlash(filepath.Dir(filename)) != dir {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ParseComments)
		if err != nil {
			return false, err
		}
		for _, name := range embeddedVariables(file) {
			if analysis.symbols[pkg.symbolPrefix()+name] {
				return true, nil
			}
		}
	}
	return false, nil
}

func embeddedVariables(file *ast.File) []string {
	var names []string
	for _, node := range file.Decls {
		group, ok := node.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || (!hasEmbedDirective(group.Doc) && !hasEmbedDirective(value.Doc)) {
				continue
			}
			for _, name := range value.Names {
				names = append(names, name.Name)
			}
		}
	}
	return names
}

func hasEmbedDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, comment := range doc.List {
		if strings.HasPrefix(comment.Text, "//go:embed") {
			return true
		}
	}
	return false
}
