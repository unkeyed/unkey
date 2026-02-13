package main

import (
	"go/ast"
	"go/token"
	"strings"
)

// findProtoImport finds the proto messages import (e.g. "github.com/.../vault/v1")
// by looking for an import that matches the module path pattern and is NOT the connect package.
func findProtoImport(f *ast.File) (alias, path string) {
	for _, imp := range f.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		// Skip stdlib, connect runtime, and connect service packages.
		if !strings.Contains(importPath, "/") ||
			importPath == "connectrpc.com/connect" ||
			strings.Contains(importPath, "net/http") ||
			strings.HasSuffix(importPath, "connect") {
			continue
		}

		if imp.Name != nil {
			return imp.Name.Name, importPath
		}
		parts := strings.Split(importPath, "/")
		return parts[len(parts)-1], importPath
	}

	return "", ""
}

// findServices extracts all ServiceClient interfaces from the AST.
func findServices(f *ast.File, protoAlias string) []serviceInfo {
	var services []serviceInfo

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !strings.HasSuffix(typeSpec.Name.Name, "ServiceClient") {
				continue
			}

			iface, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			svc := extractService(typeSpec.Name.Name, iface)
			if len(svc.Methods) > 0 {
				services = append(services, svc)
			}
		}
	}

	return services
}

// extractService extracts unary and server-streaming methods from a ServiceClient interface.
func extractService(name string, iface *ast.InterfaceType) serviceInfo {
	svc := serviceInfo{Name: name, Methods: nil}

	for _, field := range iface.Methods.List {
		if m, ok := extractMethod(field); ok {
			svc.Methods = append(svc.Methods, m)
		}
	}

	return svc
}

// extractMethod extracts a single method (unary or server-streaming) from an interface field.
// Returns false for embedded interfaces or anything that doesn't match the expected
// connect RPC method signatures.
func extractMethod(field *ast.Field) (methodInfo, bool) {
	if len(field.Names) == 0 {
		return methodInfo{Name: "", ReqType: "", RespType: "", Kind: ""}, false
	}

	funcType, ok := field.Type.(*ast.FuncType)
	if !ok {
		return methodInfo{Name: "", ReqType: "", RespType: "", Kind: ""}, false
	}

	// RPC methods have exactly 2 params (ctx, req) and 2 results (resp/stream, error).
	if funcType.Params == nil || len(funcType.Params.List) != 2 ||
		funcType.Results == nil || len(funcType.Results.List) != 2 {
		return methodInfo{Name: "", ReqType: "", RespType: "", Kind: ""}, false
	}

	retType := funcType.Results.List[0].Type
	retTypeStr := typeToString(retType)

	var kind methodKind
	switch {
	case strings.Contains(retTypeStr, "ServerStreamForClient"):
		kind = methodKindServerStream
	case strings.Contains(retTypeStr, "Response"):
		kind = methodKindUnary
	default:
		return methodInfo{Name: "", ReqType: "", RespType: "", Kind: ""}, false
	}

	reqType := extractGenericTypeArg(funcType.Params.List[1].Type)
	respType := extractGenericTypeArg(retType)
	if reqType == "" || respType == "" {
		return methodInfo{Name: "", ReqType: "", RespType: "", Kind: ""}, false
	}

	return methodInfo{
		Name:     field.Names[0].Name,
		ReqType:  reqType,
		RespType: respType,
		Kind:     kind,
	}, true
}

// extractGenericTypeArg extracts the type argument from a generic type like
// *connect.Request[v1.FooRequest] or *connect.Response[v1.FooResponse].
func extractGenericTypeArg(expr ast.Expr) string {
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
