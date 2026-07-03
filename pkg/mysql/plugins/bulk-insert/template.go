package main

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

//go:embed bulk_insert.go.tmpl
var bulkInsertTemplate string

// TemplateRenderer handles rendering of bulk insert function templates.
type TemplateRenderer struct{}

// TemplateData represents the data passed to the template.
type TemplateData struct {
	Package                   string
	BulkFunctionName          string
	BulkQueryConstant         string
	ParamsStructName          string
	InsertPart                string
	ValuesPart                string
	OnDuplicateKeyUpdate      string
	EmitMethodsWithDBArgument bool
	Fields                    []string
	ValuesFields              []string // Fields used in VALUES clause
	UpdateFields              []string // Fields used in ON DUPLICATE KEY UPDATE clause
}

// NewTemplateRenderer creates a new template renderer.
func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

// escapeBackticks escapes backticks in SQL strings for Go raw string literals.
func escapeBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "` + \"`\" + `")
}

// Render renders the bulk insert function template with the given data.
func (r *TemplateRenderer) Render(data TemplateData) (string, error) {
	// Escape backticks in SQL strings
	data.InsertPart = escapeBackticks(data.InsertPart)
	data.OnDuplicateKeyUpdate = escapeBackticks(data.OnDuplicateKeyUpdate)

	tmpl, err := template.New("bulk_insert").Parse(bulkInsertTemplate)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
