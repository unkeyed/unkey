// generate-rpc-clients generates simplified client wrappers for Connect RPC service interfaces.
//
// It parses generated *connect/*.connect.go files, extracts *ServiceClient interfaces,
// and produces wrapper packages that hide connect.Request/connect.Response boilerplate.
// Both unary and server-streaming methods are supported. Unary methods unwrap
// connect.Request/Response, while streaming methods unwrap the request but pass
// through the *connect.ServerStreamForClient as-is.
//
// Usage:
//
//	go run github.com/unkeyed/unkey/tools/generate-rpc-clients -source './vaultv1connect/*.connect.go' -out ./vaultrpc/
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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

	services := findServices(f, protoAlias)
	if len(services) == 0 {
		return nil // no unary methods to wrap
	}

	data := fileData{
		PackageName:   filepath.Base(outDir),
		ConnectPkg:    connectPkg,
		ConnectImport: protoImport + "/" + connectPkg,
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
