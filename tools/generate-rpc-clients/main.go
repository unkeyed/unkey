// generate-rpc-clients generates simplified client wrappers for Connect RPC service interfaces.
//
// It parses generated *connect/*.connect.go files, extracts *ServiceClient interfaces,
// and produces wrapper packages that hide connect.Request/connect.Response boilerplate.
// Streaming methods are skipped since they cannot use the simple wrap/unwrap pattern.
//
// Usage:
//
//	go run github.com/unkeyed/unkey/tools/generate-rpc-clients -source './vaultv1connect/*.connect.go' -out ./vaultrpc/
package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed wrapper.go.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "wrapper.go.tmpl"))

type serviceInfo struct {
	Name    string // e.g. "VaultServiceClient"
	Methods []methodInfo
}

type methodInfo struct {
	Name     string // e.g. "Encrypt"
	ReqType  string // e.g. "EncryptRequest"
	RespType string // e.g. "EncryptResponse"
}

type fileData struct {
	PackageName   string // e.g. "vaultrpc"
	ConnectPkg    string // e.g. "vaultv1connect"
	ConnectImport string // e.g. "github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	ProtoAlias    string // e.g. "v1"
	ProtoImport   string // e.g. "github.com/unkeyed/unkey/gen/proto/vault/v1"
	Services      []serviceInfo
}

func main() {
	sourceGlob := flag.String("source", "", "Glob pattern for input .connect.go files")
	outDir := flag.String("out", "", "Output directory for generated wrapper files")
	flag.Parse()

	if *sourceGlob == "" || *outDir == "" {
		log.Fatal("both -source and -out flags are required")
	}

	matches, err := filepath.Glob(*sourceGlob)
	if err != nil {
		log.Fatalf("invalid glob pattern: %v", err)
	}

	if len(matches) == 0 {
		log.Fatalf("no files matched pattern %q", *sourceGlob)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	for _, srcPath := range matches {
		if err := processFile(srcPath, *outDir); err != nil {
			log.Fatalf("processing %s: %v", srcPath, err)
		}
	}
}

func processFile(srcPath, outDir string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	connectPkg := f.Name.Name // e.g. "ctrlv1connect"

	protoAlias, protoImport := findProtoImport(f)
	if protoImport == "" {
		return fmt.Errorf("could not find proto import in %s", srcPath)
	}

	connectImport := protoImport + "/" + connectPkg

	var services []serviceInfo
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			name := typeSpec.Name.Name
			if !strings.HasSuffix(name, "ServiceClient") {
				continue
			}

			if strings.HasSuffix(name, "ServiceHandler") {
				continue
			}

			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			svc := extractService(name, iface, protoAlias)
			if len(svc.Methods) > 0 {
				services = append(services, svc)
			}
		}
	}

	if len(services) == 0 {
		return nil // no unary methods to wrap
	}

	pkgName := filepath.Base(outDir)

	data := fileData{
		PackageName:   pkgName,
		ConnectPkg:    connectPkg,
		ConnectImport: connectImport,
		ProtoAlias:    protoAlias,
		ProtoImport:   protoImport,
		Services:      services,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("template error: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt error: %w\n%s", err, buf.String())
	}

	baseName := filepath.Base(srcPath)
	outName := strings.TrimSuffix(baseName, ".connect.go") + "_generated.go"
	outPath := filepath.Join(outDir, outName)

	if err := os.WriteFile(outPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	fmt.Printf("  %s -> %s\n", srcPath, outPath)
	return nil
}

// findProtoImport finds the proto messages import (e.g. "github.com/.../vault/v1")
// by looking for an import that matches the module path pattern and is NOT the connect package.
func findProtoImport(f *ast.File) (alias, path string) {
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		if !strings.Contains(importPath, "/") {
			continue
		}
		if importPath == "connectrpc.com/connect" {
			continue
		}
		if strings.Contains(importPath, "net/http") {
			continue
		}
		if strings.HasSuffix(importPath, "connect") {
			continue
		}
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			parts := strings.Split(importPath, "/")
			alias = parts[len(parts)-1]
		}
		return alias, importPath
	}
	return "", ""
}

// extractService extracts unary methods from a ServiceClient interface.
// Streaming methods (returning *connect.ServerStreamForClient) are skipped.
func extractService(name string, iface *ast.InterfaceType, protoAlias string) serviceInfo {
	svc := serviceInfo{Name: name}

	for _, method := range iface.Methods.List {
		if len(method.Names) == 0 {
			continue
		}
		methodName := method.Names[0].Name

		funcType, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		if funcType.Results == nil || len(funcType.Results.List) != 2 {
			continue
		}

		retType := funcType.Results.List[0].Type
		retStr := typeToString(retType)
		if strings.Contains(retStr, "ServerStreamForClient") {
			continue
		}

		if funcType.Params == nil || len(funcType.Params.List) != 2 {
			continue
		}

		reqType := extractGenericTypeArg(funcType.Params.List[1].Type, protoAlias)
		respType := extractGenericTypeArg(retType, protoAlias)

		if reqType == "" || respType == "" {
			continue
		}

		svc.Methods = append(svc.Methods, methodInfo{
			Name:     methodName,
			ReqType:  reqType,
			RespType: respType,
		})
	}

	return svc
}

// extractGenericTypeArg extracts the type argument from a generic type like
// *connect.Request[v1.FooRequest] or *connect.Response[v1.FooResponse].
func extractGenericTypeArg(expr ast.Expr, protoAlias string) string {
	starExpr, ok := expr.(*ast.StarExpr)
	if !ok {
		return ""
	}

	indexExpr, ok := starExpr.X.(*ast.IndexExpr)
	if !ok {
		return ""
	}

	sel, ok := indexExpr.Index.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	return sel.Sel.Name
}

// typeToString converts an AST expression to a rough string representation
// for pattern matching (e.g., detecting ServerStreamForClient).
func typeToString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return "*" + typeToString(e.X)
	case *ast.SelectorExpr:
		return typeToString(e.X) + "." + e.Sel.Name
	case *ast.Ident:
		return e.Name
	case *ast.IndexExpr:
		return typeToString(e.X) + "[" + typeToString(e.Index) + "]"
	default:
		return ""
	}
}
