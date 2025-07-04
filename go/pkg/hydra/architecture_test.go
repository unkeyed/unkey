package hydra

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchitecturalConstraints enforces architectural rules to prevent regression
func TestArchitecturalConstraints(t *testing.T) {
	t.Run("NoGORMImports", func(t *testing.T) {
		// Scan all Go files in the hydra package for GORM imports
		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !strings.HasSuffix(path, ".go") || strings.Contains(path, "_test.go") {
				return nil
			}

			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, "\"")
				if strings.Contains(importPath, "gorm.io") {
					t.Errorf("GORM import found in %s: %s", path, importPath)
				}
			}
			return nil
		})

		if err != nil {
			t.Fatalf("Failed to walk directory: %v", err)
		}
	})

	t.Run("OnlySQLCInEngine", func(t *testing.T) {
		// Verify that Engine only references SQLC store, not GORM
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "engine.go", nil, parser.ParseComments)
		if err != nil {
			t.Fatalf("Failed to parse engine.go: %v", err)
		}

		// Check that Engine struct only has one store field
		ast.Inspect(node, func(n ast.Node) bool {
			if structType, ok := n.(*ast.StructType); ok {
				storeFields := 0
				for _, field := range structType.Fields.List {
					for _, name := range field.Names {
						if strings.Contains(name.Name, "store") || strings.Contains(name.Name, "Store") {
							storeFields++
						}
					}
				}
				if storeFields > 1 {
					t.Errorf("Engine struct should only have one store field, found %d", storeFields)
				}
			}
			return true
		})
	})
}

// TestStorageLayerSeparation ensures clean separation between storage implementations
func TestStorageLayerSeparation(t *testing.T) {
	t.Run("NoDirectSQLCImportsInBusinessLogic", func(t *testing.T) {
		// Business logic files should not import SQLC directly
		businessLogicFiles := []string{"engine.go", "worker.go", "workflow.go"}

		for _, file := range businessLogicFiles {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
			if err != nil {
				continue // Skip if file doesn't exist
			}

			for _, imp := range node.Imports {
				importPath := strings.Trim(imp.Path.Value, "\"")
				if strings.Contains(importPath, "store/sqlc") {
					t.Errorf("Direct SQLC import found in business logic file %s: %s", file, importPath)
				}
			}
		}
	})
}
