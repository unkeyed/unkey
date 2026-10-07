package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

type pageDoc struct {
	Name    string
	Source  string
	Comment string
	Fields  []fieldDoc
}

type fieldDoc struct {
	Path     string
	Heading  string
	Type     string
	Metadata []string
	Comment  string
	Source   string
}

type declaration struct {
	spec      *ast.TypeSpec
	doc       *ast.CommentGroup
	file      *ast.File
	directory string
}

type packageSource struct {
	types   map[string]declaration
	symbols map[string]token.Pos
}

type extractor struct {
	files    *token.FileSet
	root     string
	ref      string
	packages map[string]*packageSource
	visiting map[*ast.TypeSpec]bool
}

func extract(sourceFile, typeName, ref string) (pageDoc, error) {
	sourceFile, err := filepath.Abs(sourceFile)
	if err != nil {
		return pageDoc{}, err
	}
	packageDir := filepath.Dir(sourceFile)
	root, err := findRepoRoot(packageDir)
	if err != nil {
		return pageDoc{}, err
	}
	e := extractor{
		files: token.NewFileSet(), root: root, ref: ref,
		packages: make(map[string]*packageSource), visiting: make(map[*ast.TypeSpec]bool),
	}
	pkg, err := e.loadPackage(packageDir)
	if err != nil {
		return pageDoc{}, err
	}
	decl, ok := pkg.types[typeName]
	if !ok || e.files.Position(decl.spec.Pos()).Filename != sourceFile {
		return pageDoc{}, fmt.Errorf("struct %s not found in %s", typeName, sourceFile)
	}
	fields, err := e.fields(decl, "", 1)
	if err != nil {
		return pageDoc{}, err
	}
	comment, err := e.renderComment(decl, decl.doc)
	if err != nil {
		return pageDoc{}, err
	}
	return pageDoc{
		Name: typeName, Source: sourceURL(e.files, root, ref, decl.spec.Pos()),
		Comment: comment, Fields: fields,
	}, nil
}

func (e *extractor) loadPackage(directory string) (*packageSource, error) {
	if pkg, ok := e.packages[directory]; ok {
		return pkg, nil
	}
	packages, err := parser.ParseDir(e.files, directory, func(info os.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	source := &packageSource{types: make(map[string]declaration), symbols: make(map[string]token.Pos)}
	for _, pkg := range packages {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if function, ok := decl.(*ast.FuncDecl); ok {
					name := function.Name.Name
					if function.Recv != nil {
						receiver := function.Recv.List[0].Type
						if pointer, ok := receiver.(*ast.StarExpr); ok {
							receiver = pointer.X
						}
						ident, ok := receiver.(*ast.Ident)
						if !ok {
							continue
						}
						name = ident.Name + "." + name
					}
					source.addSymbol(name, function.Pos())
				}
				group, ok := decl.(*ast.GenDecl)
				if !ok {
					continue
				}
				for _, spec := range group.Specs {
					if value, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range value.Names {
							source.addSymbol(name.Name, name.Pos())
						}
					}
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if _, exists := source.types[typeSpec.Name.Name]; exists {
						return nil, fmt.Errorf("ambiguous type %s in %s", typeSpec.Name.Name, directory)
					}
					doc := typeSpec.Doc
					if doc == nil {
						doc = group.Doc
					}
					source.types[typeSpec.Name.Name] = declaration{spec: typeSpec, doc: doc, file: file, directory: directory}
					source.addSymbol(typeSpec.Name.Name, typeSpec.Pos())
					if structure, ok := typeSpec.Type.(*ast.StructType); ok {
						for _, field := range structure.Fields.List {
							for _, name := range field.Names {
								source.addSymbol(typeSpec.Name.Name+"."+name.Name, name.Pos())
							}
						}
					}
				}
			}
		}
	}
	e.packages[directory] = source
	return source, nil
}

func (p *packageSource) addSymbol(name string, pos token.Pos) {
	if _, exists := p.symbols[name]; exists {
		pos = token.NoPos
	}
	p.symbols[name] = pos
}

func (e *extractor) fields(owner declaration, prefix string, depth int) ([]fieldDoc, error) {
	structType, ok := owner.spec.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf("%s is not a struct", owner.spec.Name.Name)
	}
	if e.visiting[owner.spec] {
		return nil, fmt.Errorf("recursive config struct at %s", prefix)
	}
	e.visiting[owner.spec] = true
	defer delete(e.visiting, owner.spec)
	var fields []fieldDoc
	for _, field := range structType.Fields.List {
		if len(field.Names) != 1 || !field.Names[0].IsExported() || field.Tag == nil {
			continue
		}
		rawTag, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return nil, err
		}
		tag := reflect.StructTag(rawTag)
		name, _, _ := strings.Cut(tag.Get("toml"), ",")
		if name == "" || name == "-" {
			continue
		}
		var typ bytes.Buffer
		if err := printer.Fprint(&typ, e.files, field.Type); err != nil {
			return nil, err
		}
		nested, err := e.nestedStruct(owner, field.Type)
		if err != nil {
			return nil, err
		}
		doc := field.Doc
		commentOwner := owner
		if doc == nil {
			doc = field.Comment
		}
		if doc == nil && nested != nil {
			doc = nested.doc
			commentOwner = *nested
		}
		comment, err := e.renderComment(commentOwner, doc)
		if err != nil {
			return nil, err
		}
		fields = append(fields, fieldDoc{
			Path: prefix + name, Type: typ.String(), Metadata: tagMetadata(tag.Get("config")),
			Heading: strings.Repeat("#", min(depth, 4)),
			Comment: comment, Source: sourceURL(e.files, e.root, e.ref, field.Pos()),
		})
		if nested != nil {
			children, err := e.fields(*nested, prefix+name+".", depth+1)
			if err != nil {
				return nil, err
			}
			fields = append(fields, children...)
		}
	}
	return fields, nil
}

func (e *extractor) nestedStruct(owner declaration, expr ast.Expr) (*declaration, error) {
	var name string
	directory := owner.directory
	switch typ := expr.(type) {
	case *ast.StarExpr:
		return e.nestedStruct(owner, typ.X)
	case *ast.ArrayType:
		return e.nestedStruct(owner, typ.Elt)
	case *ast.Ident:
		name = typ.Name
	case *ast.SelectorExpr:
		qualifier, ok := typ.X.(*ast.Ident)
		if !ok {
			return nil, nil
		}
		imports, err := importPaths(owner.file)
		if err != nil {
			return nil, err
		}
		if path := imports[qualifier.Name]; strings.HasPrefix(path, "github.com/unkeyed/unkey/") {
			directory = filepath.Join(e.root, strings.TrimPrefix(path, "github.com/unkeyed/unkey/"))
			name = typ.Sel.Name
		}
	}
	if name == "" {
		return nil, nil
	}
	pkg, err := e.loadPackage(directory)
	if err != nil {
		return nil, err
	}
	decl, ok := pkg.types[name]
	if ok {
		if _, isStruct := decl.spec.Type.(*ast.StructType); isStruct {
			return &decl, nil
		}
		if e.visiting[decl.spec] {
			return nil, fmt.Errorf("recursive config type %s", name)
		}
		e.visiting[decl.spec] = true
		defer delete(e.visiting, decl.spec)
		return e.nestedStruct(decl, decl.spec.Type)
	}
	return nil, nil
}

func tagMetadata(tag string) []string {
	var metadata []string
	for _, part := range strings.Split(tag, ",") {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch name {
		case "":
			continue
		case "default":
			metadata = append(metadata, "Default: `"+value+"`")
		case "min":
			metadata = append(metadata, "Minimum: `"+value+"`")
		case "max":
			metadata = append(metadata, "Maximum: `"+value+"`")
		default:
			metadata = append(metadata, "`"+strings.TrimSpace(part)+"`")
		}
	}
	return metadata
}

func (e *extractor) renderComment(owner declaration, group *ast.CommentGroup) (string, error) {
	pkg, err := e.loadPackage(owner.directory)
	if err != nil {
		return "", err
	}
	imports, err := importPaths(owner.file)
	if err != nil {
		return "", err
	}
	parser := comment.Parser{
		Words: nil,
		LookupPackage: func(name string) (string, bool) {
			path, ok := imports[name]
			if !ok && name == owner.file.Name.Name {
				return "", true
			}
			return path, ok
		},
		LookupSym: func(recv, name string) bool {
			if recv != "" {
				name = recv + "." + name
			}
			return pkg.symbols[name].IsValid()
		},
	}
	var printer comment.Printer
	doc := parser.Parse(group.Text())
	if err := e.resolveCommentLinks(owner, doc.Content); err != nil {
		return "", err
	}
	var blocks []string
	for _, block := range doc.Content {
		if code, ok := block.(*comment.Code); ok {
			fence := "```"
			for strings.Contains(code.Text, fence) {
				fence += "`"
			}
			blocks = append(blocks, fence+"plaintext\n"+code.Text+fence)
		} else {
			part := *doc
			part.Content = []comment.Block{block}
			blocks = append(blocks, strings.TrimSpace(string(printer.Markdown(&part))))
		}
	}
	return strings.Join(blocks, "\n\n"), nil
}

func (e *extractor) resolveCommentLinks(owner declaration, blocks []comment.Block) error {
	for _, block := range blocks {
		var text *[]comment.Text
		switch block := block.(type) {
		case *comment.Paragraph:
			text = &block.Text
		case *comment.Heading:
			text = &block.Text
		case *comment.List:
			for _, item := range block.Items {
				if err := e.resolveCommentLinks(owner, item.Content); err != nil {
					return err
				}
			}
		}
		if text == nil {
			continue
		}
		var resolved []comment.Text
		for _, item := range *text {
			link, ok := item.(*comment.DocLink)
			if !ok {
				resolved = append(resolved, item)
				continue
			}
			url, err := e.docLinkURL(owner, link)
			if err != nil {
				return err
			}
			if url == "" {
				resolved = append(resolved, comment.Plain("["))
				resolved = append(resolved, link.Text...)
				resolved = append(resolved, comment.Plain("]"))
			} else {
				resolved = append(resolved, &comment.Link{Text: link.Text, URL: url, Auto: false})
			}
		}
		*text = resolved
	}
	return nil
}

func (e *extractor) docLinkURL(owner declaration, link *comment.DocLink) (string, error) {
	const module = "github.com/unkeyed/unkey"
	directory := owner.directory
	if link.ImportPath != "" {
		if link.ImportPath != module && !strings.HasPrefix(link.ImportPath, module+"/") {
			return link.DefaultURL("https://pkg.go.dev"), nil
		}
		directory = filepath.Join(e.root, strings.TrimPrefix(link.ImportPath, module))
	}
	pkg, err := e.loadPackage(directory)
	if err != nil {
		return "", err
	}
	if link.Name == "" {
		return "https://github.com/unkeyed/unkey/tree/" + e.ref + filepath.ToSlash(strings.TrimPrefix(directory, e.root)), nil
	}
	name := link.Name
	if link.Recv != "" {
		name = link.Recv + "." + name
	}
	if pos := pkg.symbols[name]; pos.IsValid() {
		return sourceURL(e.files, e.root, e.ref, pos), nil
	}
	return "", nil
}

func importPaths(file *ast.File) (map[string]string, error) {
	imports := make(map[string]string)
	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			return nil, err
		}
		name := filepath.Base(path)
		if imported.Name != nil {
			name = imported.Name.Name
		}
		imports[name] = path
	}
	return imports, nil
}

func sourceURL(files *token.FileSet, root, ref string, pos token.Pos) string {
	position := files.Position(pos)
	return fmt.Sprintf("https://github.com/unkeyed/unkey/blob/%s/%s#L%d",
		ref, filepath.ToSlash(strings.TrimPrefix(position.Filename, root+string(filepath.Separator))), position.Line)
}

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
