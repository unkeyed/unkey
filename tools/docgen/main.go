package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/doc"
	"go/doc/comment"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type meta struct {
	Title string   `json:"title"`
	Root  bool     `json:"root"`
	Pages []string `json:"pages,omitempty"`
}

var printer comment.Printer

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "usage: docgen <repo-root> <output-dir>\n")
		os.Exit(1)
	}

	repoRoot := os.Args[1]
	outDir := os.Args[2]

	if err := os.RemoveAll(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to clean output dir: %v\n", err)
		os.Exit(1)
	}

	dirs := []string{"pkg", "svc", "internal"}
	topLevel := []string{}

	for _, dir := range dirs {
		root := filepath.Join(repoRoot, dir)
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		found := false
		seen := map[string]bool{}

		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Name() != "doc.go" {
				return nil
			}

			pkgDir := filepath.Dir(path)
			if seen[pkgDir] {
				return nil
			}
			seen[pkgDir] = true

			rel, err := filepath.Rel(repoRoot, pkgDir)
			if err != nil {
				return err
			}

			if err := processPackage(pkgDir, rel, outDir); err != nil {
				fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", rel, err)
				return nil
			}

			fmt.Fprintf(os.Stderr, "  ✓ %s\n", rel)
			found = true
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error walking %s: %v\n", dir, err)
			os.Exit(1)
		}

		if found {
			topLevel = append(topLevel, dir)
		}
	}

	if err := generateAllMeta(outDir, topLevel); err != nil {
		fmt.Fprintf(os.Stderr, "error generating meta.json: %v\n", err)
		os.Exit(1)
	}

	if err := collapseLeafDirs(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "error collapsing leaf dirs: %v\n", err)
		os.Exit(1)
	}

	if err := generateIndex(outDir); err != nil {
		fmt.Fprintf(os.Stderr, "error generating index: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "done\n")
}

func processPackage(pkgDir, relDir, outDir string) error {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, pkgDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse dir: %w", err)
	}

	for _, pkg := range pkgs {
		dpkg := doc.New(pkg, relDir, doc.AllDecls)

		if dpkg.Doc == "" {
			return fmt.Errorf("no package doc comment")
		}

		title := makeTitle(relDir)
		description := extractDescription(dpkg.Doc)

		destDir := filepath.Join(outDir, relDir)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		var buf strings.Builder
		buf.WriteString("---\n")
		buf.WriteString(fmt.Sprintf("title: %s\n", title))
		buf.WriteString(fmt.Sprintf("description: %q\n", description))
		buf.WriteString("---\n\n")

		buf.Write(docToMarkdown(dpkg.Doc))

		writeConstants(&buf, fset, dpkg.Consts)
		writeVariables(&buf, fset, dpkg.Vars)
		writeFunctions(&buf, fset, dpkg.Funcs)
		writeTypes(&buf, fset, dpkg.Types)

		destFile := filepath.Join(destDir, "index.md")
		if err := os.WriteFile(destFile, []byte(buf.String()), 0o644); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("no packages found")
}

var headerAnchorRe = regexp.MustCompile(`( \{#hdr-[^}]+\})`)

func docToMarkdown(text string) []byte {
	var p comment.Parser
	d := p.Parse(text)
	md := printer.Markdown(d)
	return headerAnchorRe.ReplaceAll(md, nil)
}

func writeConstants(buf *strings.Builder, fset *token.FileSet, consts []*doc.Value) {
	if len(consts) == 0 {
		return
	}

	buf.WriteString("\n## Constants\n\n")
	for _, c := range consts {
		if c.Doc != "" {
			buf.Write(docToMarkdown(c.Doc))
		}
		buf.WriteString("```go\n")
		buf.WriteString(formatDecl(fset, c.Decl))
		buf.WriteString("\n```\n\n")
	}
}

func writeVariables(buf *strings.Builder, fset *token.FileSet, vars []*doc.Value) {
	if len(vars) == 0 {
		return
	}

	buf.WriteString("\n## Variables\n\n")
	for _, v := range vars {
		if v.Doc != "" {
			buf.Write(docToMarkdown(v.Doc))
		}
		buf.WriteString("```go\n")
		buf.WriteString(formatDecl(fset, v.Decl))
		buf.WriteString("\n```\n\n")
	}
}

func writeFunctions(buf *strings.Builder, fset *token.FileSet, funcs []*doc.Func) {
	if len(funcs) == 0 {
		return
	}

	buf.WriteString("\n## Functions\n\n")
	for _, f := range funcs {
		if !ast.IsExported(f.Name) {
			continue
		}
		buf.WriteString(fmt.Sprintf("### func %s\n\n", f.Name))
		buf.WriteString("```go\n")
		buf.WriteString(formatDecl(fset, f.Decl))
		buf.WriteString("\n```\n\n")
		if f.Doc != "" {
			buf.Write(docToMarkdown(f.Doc))
			buf.WriteString("\n")
		}
	}
}

func writeTypes(buf *strings.Builder, fset *token.FileSet, types []*doc.Type) {
	if len(types) == 0 {
		return
	}

	buf.WriteString("\n## Types\n\n")
	for _, t := range types {
		if !ast.IsExported(t.Name) {
			continue
		}
		buf.WriteString(fmt.Sprintf("### type %s\n\n", t.Name))
		buf.WriteString("```go\n")
		buf.WriteString(formatDecl(fset, t.Decl))
		buf.WriteString("\n```\n\n")
		if t.Doc != "" {
			buf.Write(docToMarkdown(t.Doc))
			buf.WriteString("\n")
		}

		for _, c := range t.Consts {
			if c.Doc != "" {
				buf.Write(docToMarkdown(c.Doc))
			}
			buf.WriteString("```go\n")
			buf.WriteString(formatDecl(fset, c.Decl))
			buf.WriteString("\n```\n\n")
		}

		for _, f := range t.Funcs {
			if !ast.IsExported(f.Name) {
				continue
			}
			buf.WriteString(fmt.Sprintf("#### func %s\n\n", f.Name))
			buf.WriteString("```go\n")
			buf.WriteString(formatDecl(fset, f.Decl))
			buf.WriteString("\n```\n\n")
			if f.Doc != "" {
				buf.Write(docToMarkdown(f.Doc))
				buf.WriteString("\n")
			}
		}

		for _, m := range t.Methods {
			if !ast.IsExported(m.Name) {
				continue
			}
			buf.WriteString(fmt.Sprintf("#### func (%s) %s\n\n", t.Name, m.Name))
			buf.WriteString("```go\n")
			buf.WriteString(formatDecl(fset, m.Decl))
			buf.WriteString("\n```\n\n")
			if m.Doc != "" {
				buf.Write(docToMarkdown(m.Doc))
				buf.WriteString("\n")
			}
		}
	}
}

func formatDecl(fset *token.FileSet, decl ast.Node) string {
	var buf strings.Builder
	if err := format.Node(&buf, fset, decl); err != nil {
		return "// failed to format declaration"
	}
	return buf.String()
}

func makeTitle(relPath string) string {
	return filepath.Base(relPath)
}

func extractDescription(docText string) string {
	firstLine := strings.SplitN(docText, "\n", 2)[0]
	firstLine = strings.TrimSpace(firstLine)

	parts := strings.SplitN(firstLine, " ", 3)
	if len(parts) >= 3 && parts[0] == "Package" {
		firstLine = parts[2]
	}

	firstLine = strings.TrimSuffix(firstLine, ".")
	return firstLine
}

func generateAllMeta(outDir string, topLevel []string) error {
	sort.Strings(topLevel)
	rootPages := append([]string{"index"}, topLevel...)
	if err := writeMeta(outDir, "Packages", true, rootPages); err != nil {
		return err
	}

	titles := map[string]string{
		"pkg":      "Packages",
		"svc":      "Services",
		"internal": "Internal",
	}

	return filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == outDir {
			return err
		}

		children := listChildDirs(path)
		if len(children) == 0 {
			return nil
		}

		rel, _ := filepath.Rel(outDir, path)
		title, ok := titles[rel]
		if !ok {
			title = makeTitle(rel)
		}

		return writeMeta(path, title, false, children)
	})
}

func listChildDirs(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs
}

func collapseLeafDirs(outDir string) error {
	var leaves []string
	err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() || path == outDir {
			return err
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		hasSubdir := false
		hasIndex := false
		fileCount := 0
		for _, e := range entries {
			if e.IsDir() {
				hasSubdir = true
				break
			}
			if e.Name() == "index.md" {
				hasIndex = true
			}
			fileCount++
		}

		if !hasSubdir && hasIndex && fileCount == 1 {
			leaves = append(leaves, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Sort(sort.Reverse(sort.StringSlice(leaves)))

	for _, dir := range leaves {
		indexFile := filepath.Join(dir, "index.md")
		flatFile := dir + ".md"

		data, err := os.ReadFile(indexFile)
		if err != nil {
			return err
		}

		if err := os.WriteFile(flatFile, data, 0o644); err != nil {
			return err
		}

		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}

	return nil
}

func generateIndex(outDir string) error {
	content := `---
title: Packages
description: Auto-generated documentation for Go packages and services.
---

Browse the sidebar to explore package documentation generated from Go source files.
`
	return os.WriteFile(filepath.Join(outDir, "index.md"), []byte(content), 0o644)
}

func writeMeta(dir, title string, root bool, pages []string) error {
	m := meta{
		Title: title,
		Root:  root,
		Pages: pages,
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "meta.json"), append(data, '\n'), 0o644)
}
