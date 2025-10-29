package main

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

// Generator handles the generation of bulk insert functions.
type Generator struct {
	options Options
}

// Options contains configuration for the bulk insert generator.
type Options struct {
	Package                   string
	EmitMethodsWithDBArgument bool
}

// BulkFunction represents a bulk insert function signature.
type BulkFunction struct {
	Name         string
	ParamsStruct string
}

// NewGenerator creates a new Generator with default options.
func NewGenerator() *Generator {
	return &Generator{
		options: Options{
			Package:                   "db",
			EmitMethodsWithDBArgument: true,
		},
	}
}

// Generate processes the plugin request and generates bulk insert functions.
func (g *Generator) Generate(req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	var files []*plugin.File

	// Collect all bulk functions for interface generation
	var bulkFunctions []BulkFunction

	// Generate bulk insert functions for each existing insert function
	for _, query := range req.GetQueries() {
		if !g.isInsertQuery(query) {
			continue
		}

		bulkFunction := g.generateBulkInsertFunction(query)
		if bulkFunction != nil {
			files = append(files, bulkFunction)

			// Generate pluralized function name for interface
			bulkFunctionName := query.GetName()
			entityName, isInsert := strings.CutPrefix(query.GetName(), "Insert")
			if isInsert {
				pluralizedEntity := pluralize(entityName)
				bulkFunctionName = "Insert" + pluralizedEntity
			}

			// Track function signature for interface
			bulkFunctions = append(bulkFunctions, BulkFunction{
				Name:         bulkFunctionName,
				ParamsStruct: fmt.Sprintf("%sParams", query.GetName()),
			})
		}
	}

	// Generate interface extension file
	if len(bulkFunctions) > 0 {
		interfaceFile := g.generateInterfaceExtension(bulkFunctions)
		if interfaceFile != nil {
			files = append(files, interfaceFile)
		}
	}

	return &plugin.GenerateResponse{
		Files: files,
	}, nil
}

// isInsertQuery checks if a query is an INSERT query that should have a bulk function generated.
func (g *Generator) isInsertQuery(query *plugin.Query) bool {
	return query.GetInsertIntoTable() != nil && len(query.GetParams()) > 0
}

// generateBulkInsertFunction generates a bulk insert function for the given query.
func (g *Generator) generateBulkInsertFunction(query *plugin.Query) *plugin.File {
	originalName := query.GetName()
	paramsStructName := fmt.Sprintf("%sParams", originalName)

	// Generate pluralized function name
	// If the function name starts with "Insert", replace it with pluralized version
	bulkFunctionName := originalName
	if entityName, ok := strings.CutPrefix(originalName, "Insert"); ok {
		pluralizedEntity := pluralize(entityName)
		bulkFunctionName = "Insert" + pluralizedEntity
	}

	// Parse the SQL query
	parser := NewSQLParser()
	parsedQuery := parser.Parse(query)

	// Extract field names from query parameters
	fields := g.extractFieldNames(query.GetParams())

	// Determine which fields are used in VALUES clause vs ON DUPLICATE KEY UPDATE
	// sqlc guarantees that parameters are ordered as they appear in the SQL query.
	// Since VALUES clause always comes before ON DUPLICATE KEY UPDATE in SQL,
	// we can safely use the placeholder count from the parsed VALUES clause
	// to determine which parameters belong to which part.
	valuesFields := fields
	var updateFields []string

	if parsedQuery.ValuesPlaceholderCount > 0 && parsedQuery.ValuesPlaceholderCount < len(fields) {
		// Split fields based on the actual number of placeholders in the VALUES clause
		valuesFields = fields[:parsedQuery.ValuesPlaceholderCount]
		// The remaining fields are for ON DUPLICATE KEY UPDATE
		updateFields = fields[parsedQuery.ValuesPlaceholderCount:]
	}

	// Generate the bulk insert function content
	renderer := NewTemplateRenderer()
	content, err := renderer.Render(TemplateData{
		Package:                   g.options.Package,
		BulkFunctionName:          bulkFunctionName,
		BulkQueryConstant:         fmt.Sprintf("bulk%s", originalName),
		ParamsStructName:          paramsStructName,
		InsertPart:                parsedQuery.InsertPart,
		ValuesPart:                parsedQuery.ValuesPart,
		OnDuplicateKeyUpdate:      parsedQuery.OnDuplicateKeyUpdate,
		EmitMethodsWithDBArgument: g.options.EmitMethodsWithDBArgument,
		Fields:                    fields,
		ValuesFields:              valuesFields,
		UpdateFields:              updateFields,
	})
	if err != nil {
		return nil
	}

	// Generate filename
	filename := fmt.Sprintf("bulk_%s_generated.go", query.GetFilename())

	return &plugin.File{
		Name:     filename,
		Contents: []byte(content),
	}
}

// extractFieldNames extracts Go field names from query parameters.
func (g *Generator) extractFieldNames(params []*plugin.Parameter) []string {
	var fields []string

	for _, param := range params {
		if param.GetColumn() != nil {
			fieldName := ToCamelCase(param.GetColumn().GetName())
			fields = append(fields, fieldName)
		}
	}

	return fields
}

// generateInterfaceExtension generates an interface extension file with bulk insert methods.
func (g *Generator) generateInterfaceExtension(bulkFunctions []BulkFunction) *plugin.File {
	content := g.generateInterfaceContent(bulkFunctions)
	return &plugin.File{
		Name:     "querier_bulk_generated.go",
		Contents: []byte(content),
	}
}

// generateInterfaceContent generates the content for the interface extension.
func (g *Generator) generateInterfaceContent(bulkFunctions []BulkFunction) string {
	var content strings.Builder

	content.WriteString("// Code generated by sqlc bulk insert plugin. DO NOT EDIT.\n\n")
	content.WriteString(fmt.Sprintf("package %s\n\n", g.options.Package))
	content.WriteString("import \"context\"\n\n")

	// Generate BulkQuerier interface
	content.WriteString("// BulkQuerier contains bulk insert methods.\n")
	content.WriteString("type BulkQuerier interface {\n")

	for _, fn := range bulkFunctions {
		if g.options.EmitMethodsWithDBArgument {
			content.WriteString(fmt.Sprintf("\t%s(ctx context.Context, db DBTX, args []%s) error\n", fn.Name, fn.ParamsStruct))
		} else {
			content.WriteString(fmt.Sprintf("\t%s(ctx context.Context, args []%s) error\n", fn.Name, fn.ParamsStruct))
		}
	}

	content.WriteString("}\n\n")

	// Generate assertion to ensure BulkQueries implements BulkQuerier
	content.WriteString("// Ensure BulkQueries implements BulkQuerier\n")
	content.WriteString("var _ BulkQuerier = (*BulkQueries)(nil)\n")

	return content.String()
}
