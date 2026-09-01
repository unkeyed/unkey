// Command configdocs generates a Mintlify configuration reference page from a
// service's Go config struct.
//
// It parses the service package and pkg/config with go/parser, walks the root
// struct's fields, and renders one ResponseField per TOML key. The toml tags
// name the keys, the config tags supply defaults and constraints, and the Go
// doc comments supply the descriptions. Improving a generated description
// therefore means improving the code comment, which keeps one source of truth.
//
// The tool resolves nested struct types syntactically instead of type-checking
// the package. This keeps it dependency-free and fast, at the cost of only
// following named struct types declared in the service package itself or in
// pkg/config. Fields of any other named type render as opaque values.
//
// Run it from the service package directory, typically via a go:generate
// directive next to the config struct:
//
//	//go:generate go run github.com/unkeyed/unkey/tools/configdocs -service logdrain -out ../../docs/...
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

func main() {
	pkgDir := flag.String("pkg", ".", "directory of the service package")
	typeName := flag.String("type", "Config", "root config struct name")
	service := flag.String("service", "", "service name used in the page prose")
	out := flag.String("out", "", "output MDX file path")
	flag.Parse()
	if *service == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "configdocs: -service and -out are required")
		os.Exit(2)
	}
	if err := run(*pkgDir, *typeName, *service, *out); err != nil {
		fmt.Fprintf(os.Stderr, "configdocs: %v\n", err)
		os.Exit(1)
	}
}

func run(pkgDir, typeName, service, out string) error {
	repoRoot, err := findRepoRoot(pkgDir)
	if err != nil {
		return err
	}
	servicePkg, err := parsePackage(pkgDir)
	if err != nil {
		return err
	}
	configPkg, err := parsePackage(filepath.Join(repoRoot, "pkg", "config"))
	if err != nil {
		return err
	}
	resolver := &resolver{service: servicePkg, config: configPkg}
	root, ok := servicePkg.structs[typeName]
	if !ok {
		return fmt.Errorf("type %s not found in %s", typeName, pkgDir)
	}
	fields, err := resolver.walk(servicePkg, root, "")
	if err != nil {
		return err
	}
	sourceFile, err := filepath.Abs(servicePkg.declFile[typeName])
	if err != nil {
		return err
	}
	sourceRel, err := filepath.Rel(repoRoot, sourceFile)
	if err != nil {
		return err
	}
	page := renderPage(service, filepath.ToSlash(sourceRel), fields)
	return os.WriteFile(out, []byte(page), 0o644)
}

// findRepoRoot walks up from dir until it finds go.mod.
func findRepoRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for current := abs; ; current = filepath.Dir(current) {
		if _, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil {
			return current, nil
		}
		if filepath.Dir(current) == current {
			return "", fmt.Errorf("go.mod not found above %s", abs)
		}
	}
}

// pkg indexes one parsed package: its struct declarations, the file each type
// was declared in, the doc comment of each type, and the package's imports.
type pkg struct {
	structs  map[string]*ast.StructType
	declFile map[string]string
	typeDoc  map[string]string
	imports  map[string]string
}

// parsePackage parses every non-test Go file in dir with comments attached.
func parsePackage(dir string) (*pkg, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", dir, err)
	}
	result := &pkg{
		structs:  map[string]*ast.StructType{},
		declFile: map[string]string{},
		typeDoc:  map[string]string{},
		imports:  map[string]string{},
	}
	for _, astPkg := range parsed {
		for fileName, file := range astPkg.Files {
			result.indexFile(fset, fileName, file)
		}
	}
	return result, nil
}

// indexFile records the file's struct declarations and import aliases.
func (p *pkg) indexFile(fset *token.FileSet, fileName string, file *ast.File) {
	for _, imported := range file.Imports {
		path := strings.Trim(imported.Path.Value, `"`)
		name := path[strings.LastIndex(path, "/")+1:]
		if imported.Name != nil {
			name = imported.Name.Name
		}
		p.imports[name] = path
	}
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, isType := spec.(*ast.TypeSpec)
			if !isType {
				continue
			}
			structType, isStruct := typeSpec.Type.(*ast.StructType)
			if !isStruct {
				continue
			}
			p.structs[typeSpec.Name.Name] = structType
			p.declFile[typeSpec.Name.Name] = fset.Position(typeSpec.Pos()).Filename
			if genDecl.Doc != nil {
				p.typeDoc[typeSpec.Name.Name] = genDecl.Doc.Text()
			} else if typeSpec.Doc != nil {
				p.typeDoc[typeSpec.Name.Name] = typeSpec.Doc.Text()
			}
		}
	}
}

// fieldDoc is one rendered configuration key or section.
type fieldDoc struct {
	Path        string
	Type        string
	Doc         string
	Default     string
	Required    bool
	Constraints []string
	Children    []fieldDoc
}

// resolver follows named struct types across the two parsed packages.
type resolver struct {
	service *pkg
	config  *pkg
}

const configImportPath = "github.com/unkeyed/unkey/pkg/config"

// walk converts one struct's fields into fieldDocs in declaration order.
func (r *resolver) walk(owner *pkg, structType *ast.StructType, prefix string) ([]fieldDoc, error) {
	var fields []fieldDoc
	for _, field := range structType.Fields.List {
		if len(field.Names) != 1 || field.Tag == nil {
			continue
		}
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		tomlName, _, _ := strings.Cut(tag.Get("toml"), ",")
		if tomlName == "" || tomlName == "-" {
			continue
		}
		doc := fieldDoc{
			Path:        joinPath(prefix, tomlName),
			Type:        "",
			Doc:         flattenDoc(docText(field)),
			Default:     "",
			Required:    false,
			Constraints: nil,
			Children:    nil,
		}
		applyConfigTag(&doc, tag.Get("config"))
		if err := r.resolveType(owner, field.Type, &doc); err != nil {
			return nil, fmt.Errorf("field %s: %w", doc.Path, err)
		}
		fields = append(fields, doc)
	}
	return fields, nil
}

// resolveType fills in the field's rendered type and, for struct types, its
// children. Unknown named types render as opaque values instead of failing so
// a new config field never blocks docs generation.
func (r *resolver) resolveType(owner *pkg, expr ast.Expr, doc *fieldDoc) error {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return r.resolveType(owner, t.X, doc)
	case *ast.Ident:
		if leaf, ok := leafType(t.Name); ok {
			doc.Type = leaf
			return nil
		}
		if nested, ok := owner.structs[t.Name]; ok {
			return r.nest(owner, t.Name, nested, doc)
		}
		doc.Type = "value"
		return nil
	case *ast.SelectorExpr:
		local, ok := t.X.(*ast.Ident)
		if !ok {
			doc.Type = "value"
			return nil
		}
		if local.Name == "time" && t.Sel.Name == "Duration" {
			doc.Type = "duration"
			return nil
		}
		if owner.imports[local.Name] == configImportPath {
			if nested, found := r.config.structs[t.Sel.Name]; found {
				return r.nest(r.config, t.Sel.Name, nested, doc)
			}
		}
		doc.Type = "value"
		return nil
	case *ast.ArrayType:
		doc.Type = "array"
		return nil
	case *ast.MapType:
		doc.Type = "map"
		return nil
	default:
		doc.Type = "value"
		return nil
	}
}

// nest renders a struct-typed field as an object section. A field without its
// own doc comment inherits the doc comment of the struct type it references.
func (r *resolver) nest(owner *pkg, typeName string, structType *ast.StructType, doc *fieldDoc) error {
	doc.Type = "object"
	if doc.Doc == "" {
		doc.Doc = flattenDoc(owner.typeDoc[typeName])
	}
	children, err := r.walk(owner, structType, doc.Path)
	if err != nil {
		return err
	}
	doc.Children = children
	return nil
}

// applyConfigTag translates config tag directives into rendered attributes.
func applyConfigTag(doc *fieldDoc, tag string) {
	for _, part := range strings.Split(tag, ",") {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch name {
		case "required":
			doc.Required = true
		case "default":
			doc.Default = value
		case "nonempty":
			doc.Constraints = append(doc.Constraints, "Must not be empty.")
		case "min":
			doc.Constraints = append(doc.Constraints, fmt.Sprintf("Minimum: %s.", value))
		case "max":
			doc.Constraints = append(doc.Constraints, fmt.Sprintf("Maximum: %s.", value))
		case "oneof":
			options := strings.Split(value, "|")
			doc.Constraints = append(doc.Constraints,
				fmt.Sprintf("Must be one of `%s`.", strings.Join(options, "`, `")))
		}
	}
}

// leafType maps Go basic type names to the rendered type label.
func leafType(name string) (string, bool) {
	switch name {
	case "string":
		return "string", true
	case "bool":
		return "bool", true
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "int", true
	case "float32", "float64":
		return "float", true
	default:
		return "", false
	}
}

// docText prefers the field's doc comment and falls back to its line comment.
func docText(field *ast.Field) string {
	if field.Doc != nil {
		return field.Doc.Text()
	}
	if field.Comment != nil {
		return field.Comment.Text()
	}
	return ""
}

// flattenDoc collapses a Go doc comment into one MDX-safe paragraph.
func flattenDoc(text string) string {
	escaped := strings.NewReplacer(
		"<", "&lt;",
		"{", "&#123;",
		"}", "&#125;",
	).Replace(text)
	return strings.Join(strings.Fields(escaped), " ")
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// renderPage produces the complete MDX document.
func renderPage(service, sourceRel string, fields []fieldDoc) string {
	var b strings.Builder
	fmt.Fprintf(&b, `---
title: "Configuration"
description: "Configuration reference for the %s service"
---

{/* Code generated by tools/configdocs. DO NOT EDIT. */}
{/* Descriptions come from the Go doc comments in %s. Edit those and run `+"`mise run generate`"+`. */}

import ConfigToml from "/snippets/config-toml.mdx";

<ConfigToml />

The schema maps to [`+"`%s`"+`](https://github.com/unkeyed/unkey/blob/main/%s).

`, service, sourceRel, sourceRel, sourceRel)
	for _, field := range fields {
		renderField(&b, field, 0)
	}
	return b.String()
}

// renderField writes one ResponseField, nesting children in an Expandable.
func renderField(b *strings.Builder, field fieldDoc, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(b, "%s<ResponseField name=%q type=%q", indent, field.Path, field.Type)
	if field.Default != "" {
		fmt.Fprintf(b, " default=%q", field.Default)
	}
	if field.Required {
		b.WriteString(" required")
	}
	b.WriteString(">\n")
	description := strings.TrimSpace(field.Doc + " " + strings.Join(field.Constraints, " "))
	if description != "" {
		fmt.Fprintf(b, "%s  %s\n", indent, description)
	}
	if len(field.Children) > 0 {
		fmt.Fprintf(b, "%s  <Expandable title=\"Fields\">\n", indent)
		for _, child := range field.Children {
			renderField(b, child, depth+2)
		}
		fmt.Fprintf(b, "%s  </Expandable>\n", indent)
	}
	fmt.Fprintf(b, "%s</ResponseField>\n", indent)
	if depth == 0 {
		b.WriteString("\n")
	}
}
