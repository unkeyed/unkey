package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

func main() {
	pkg := flag.String("pkg", "", "path to Go package directory (e.g. ./svc/api)")
	typeName := flag.String("type", "Config", "root struct type name")
	out := flag.String("out", "", "output file path (stdout if empty)")
	title := flag.String("title", "", "MDX frontmatter title (defaults to package name)")
	flag.Parse()

	if *pkg == "" {
		fmt.Fprintln(os.Stderr, "usage: generate-config-docs --pkg=./svc/api [--type=Config] [--out=path.mdx] [--title=API]")
		os.Exit(1)
	}

	absDir, err := filepath.Abs(*pkg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve path: %v\n", err)
		os.Exit(1)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, absDir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse package: %v\n", err)
		os.Exit(1)
	}

	// Collect all files across packages in this directory.
	var allFiles []*ast.File
	for _, p := range pkgs {
		for _, f := range p.Files {
			allFiles = append(allFiles, f)
		}
	}

	if len(allFiles) == 0 {
		fmt.Fprintf(os.Stderr, "no Go files in %s\n", absDir)
		os.Exit(1)
	}

	// Build a map of type name → struct fields across all files.
	types := collectStructTypes(allFiles)

	root, ok := types[*typeName]
	if !ok {
		fmt.Fprintf(os.Stderr, "type %q not found in %s\n", *typeName, absDir)
		os.Exit(1)
	}

	sections := buildSections(root, types)

	pageTitle := *title
	if pageTitle == "" {
		// Derive from directory name.
		pageTitle = strings.Title(filepath.Base(absDir)) //nolint:staticcheck
	}

	mdx, err := renderMDX(pageTitle, sections)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}

	if *out == "" {
		fmt.Print(mdx)
		return
	}

	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, []byte(mdx), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

// structInfo holds parsed information about a struct type.
type structInfo struct {
	doc    string // doc comment on the type
	fields []fieldInfo
}

// fieldInfo holds parsed information about a single struct field.
type fieldInfo struct {
	goName    string
	tomlKey   string
	goType    string
	doc       string
	required  bool
	nonempty  bool
	defVal    string
	min       string
	max       string
	oneof     string
	isStruct  bool
	structRef string // type name when isStruct
	skip      bool   // toml:"-"
}

// collectStructTypes walks all files and extracts struct type declarations.
func collectStructTypes(files []*ast.File) map[string]*structInfo {
	types := make(map[string]*structInfo)

	for _, file := range files {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					continue
				}

				doc := ""
				if ts.Doc != nil {
					doc = cleanComment(ts.Doc.Text())
				} else if gd.Doc != nil {
					doc = cleanComment(gd.Doc.Text())
				}

				var fields []fieldInfo
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						continue // embedded
					}
					fields = append(fields, parseField(field))
				}

				types[ts.Name.Name] = &structInfo{
					doc:    doc,
					fields: fields,
				}
			}
		}
	}

	return types
}

// parseField extracts field metadata from an AST field.
func parseField(field *ast.Field) fieldInfo {
	fi := fieldInfo{
		goName:    field.Names[0].Name,
		tomlKey:   "",
		goType:    exprToString(field.Type),
		doc:       "",
		required:  false,
		nonempty:  false,
		defVal:    "",
		min:       "",
		max:       "",
		oneof:     "",
		isStruct:  false,
		structRef: "",
		skip:      false,
	}

	if field.Doc != nil {
		fi.doc = cleanComment(field.Doc.Text())
	}

	if field.Tag != nil {
		tagValue := field.Tag.Value
		// Strip backticks.
		tagValue = strings.Trim(tagValue, "`")

		fi.tomlKey = lookupTag(tagValue, "toml")
		if fi.tomlKey == "-" {
			fi.skip = true
			return fi
		}

		configTag := lookupTag(tagValue, "config")
		if configTag != "" {
			fi.parseConfigTag(configTag)
		}
	}

	// Detect struct references.
	typeName := fi.goType
	if strings.HasPrefix(typeName, "*") {
		typeName = typeName[1:]
	}
	// Simple heuristic: if it starts with uppercase and isn't a builtin, it's
	// likely a struct type defined in this package.
	if len(typeName) > 0 && typeName[0] >= 'A' && typeName[0] <= 'Z' &&
		!isBuiltinType(typeName) {
		fi.isStruct = true
		fi.structRef = typeName
	}

	return fi
}

func (fi *fieldInfo) parseConfigTag(tag string) {
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		name, value, _ := strings.Cut(part, "=")
		switch name {
		case "required":
			fi.required = true
		case "nonempty":
			fi.nonempty = true
		case "default":
			fi.defVal = value
		case "min":
			fi.min = value
		case "max":
			fi.max = value
		case "oneof":
			fi.oneof = value
		}
	}
}

// lookupTag extracts a struct tag value, mimicking reflect.StructTag.Get.
func lookupTag(raw, key string) string {
	tag := reflect.StructTag(raw)
	v := tag.Get(key)
	if v == "" {
		return ""
	}
	// For toml/json/yaml tags, take only the part before the first comma.
	if key == "toml" || key == "json" || key == "yaml" {
		v, _, _ = strings.Cut(v, ",")
	}
	return v
}

func exprToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	case *ast.MapType:
		return "map[" + exprToString(e.Key) + "]" + exprToString(e.Value)
	default:
		return "unknown"
	}
}

func isBuiltinType(name string) bool {
	switch name {
	case "Duration":
		return true
	default:
		return false
	}
}

func cleanComment(s string) string {
	return strings.TrimSpace(s)
}

// Section represents a group of config properties, either top-level fields
// or the fields of a nested struct.
type Section struct {
	Title       string
	Description string
	TOMLPrefix  string
	Properties  []Property
}

// Property is the template-friendly representation of a config field.
type Property struct {
	TOMLKey     string
	DisplayType string
	Description string
	Required    bool
	Default     string
	Constraints string // e.g. "min=0, max=1"
	OneOf       string
}

// buildSections converts the root struct into a flat list of sections.
func buildSections(root *structInfo, types map[string]*structInfo) []Section {
	var sections []Section

	// Top-level scalar fields go into the first section.
	topSection := Section{
		Title:       "General",
		Description: "",
		TOMLPrefix:  "",
		Properties:  nil,
	}

	for _, f := range root.fields {
		if f.skip {
			continue
		}
		if f.isStruct {
			// Nested struct → separate section.
			sub, ok := types[f.structRef]
			if !ok {
				continue
			}
			sec := Section{
				Title:       structNameToTitle(f.structRef),
				Description: sub.doc,
				TOMLPrefix:  f.tomlKey,
				Properties:  nil,
			}
			for _, sf := range sub.fields {
				if sf.skip {
					continue
				}
				sec.Properties = append(sec.Properties, toProperty(sf, f.tomlKey))
			}
			sections = append(sections, sec)
		} else {
			topSection.Properties = append(topSection.Properties, toProperty(f, ""))
		}
	}

	// Prepend the top-level section if it has properties.
	if len(topSection.Properties) > 0 {
		sections = append([]Section{topSection}, sections...)
	}

	return sections
}

func toProperty(f fieldInfo, prefix string) Property {
	key := f.tomlKey
	if key == "" {
		key = strings.ToLower(f.goName)
	}
	if prefix != "" {
		key = prefix + "." + key
	}

	p := Property{
		TOMLKey:     key,
		DisplayType: goTypeToDisplay(f.goType),
		Description: f.doc,
		Required:    f.required || f.nonempty,
		Default:     f.defVal,
		Constraints: "",
		OneOf:       "",
	}

	var constraints []string
	if f.min != "" {
		constraints = append(constraints, "min: "+f.min)
	}
	if f.max != "" {
		constraints = append(constraints, "max: "+f.max)
	}
	if len(constraints) > 0 {
		p.Constraints = strings.Join(constraints, ", ")
	}
	if f.oneof != "" {
		p.OneOf = strings.ReplaceAll(f.oneof, "|", ", ")
	}

	return p
}

func goTypeToDisplay(t string) string {
	switch t {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64":
		return "integer"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "time.Duration":
		return "duration"
	case "[]string":
		return "string[]"
	default:
		if strings.HasPrefix(t, "[]") {
			return strings.TrimPrefix(t, "[]") + "[]"
		}
		return t
	}
}

// compoundWords maps CamelCase identifiers to their preferred display form
// so that structNameToTitle does not split known compound words.
var compoundWords = map[string]string{
	"ClickHouse": "ClickHouse",
}

func structNameToTitle(name string) string {
	// Remove "Config" suffix for cleaner titles.
	name = strings.TrimSuffix(name, "Config")
	if name == "" {
		return "Configuration"
	}

	if override, ok := compoundWords[name]; ok {
		return override
	}

	// Insert spaces before uppercase letters, but keep consecutive uppercase
	// letters together (e.g. "TLS" stays "TLS", "TLSFiles" → "TLS Files").
	var b strings.Builder
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'
			prevIsLower := prev >= 'a' && prev <= 'z'
			if prevIsLower || (prev >= 'A' && prev <= 'Z' && nextIsLower) {
				b.WriteRune(' ')
			}
		}
		b.WriteRune(r)
	}
	return b.String()
}

// renderMDX executes the template and returns the MDX string.
func renderMDX(title string, sections []Section) (string, error) {
	tmpl, err := template.New("config-docs").Funcs(template.FuncMap{
		"quote": func(s string) string { return strconv.Quote(s) },
	}).Parse(mdxTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	data := struct {
		Title    string
		Sections []Section
	}{
		Title:    title,
		Sections: sections,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

var mdxTemplate = `---
title: {{ .Title }} Configuration
---
import {Property} from "fumadocs-openapi/ui"

{/* Auto-generated from Go source. Do not edit manually. */}
{/* Run: go run ./cmd/generate-config-docs --pkg=./svc/<service> --type=Config */}

This page documents all configuration options available for the {{ .Title }} service.
Configuration is loaded from a TOML file. Environment variables are expanded
using ` + "`${VAR}`" + ` or ` + "`${VAR:-default}`" + ` syntax before parsing.

{{ range .Sections }}
## {{ .Title }}
{{ if .Description }}
{{ .Description }}
{{ end }}
{{ range .Properties }}
<Property name={{ quote .TOMLKey }} type={{ quote .DisplayType }}{{ if .Default }} defaultValue={{ quote .Default }}{{ end }} required={{ "{" }}{{ .Required }}{{ "}" }}>
  {{ .Description }}
{{ if .Constraints }}
  **Constraints:** {{ .Constraints }}
{{ end }}{{ if .OneOf }}
  **Allowed values:** ` + "`" + `{{ .OneOf }}` + "`" + `
{{ end }}
</Property>
{{ end }}
{{ end }}`
