package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

//go:embed bulk_update.go.tmpl
var bulkUpdateTemplate string

const updateIdentifier = "(?:[a-zA-Z_][a-zA-Z0-9_]*|`[a-zA-Z_][a-zA-Z0-9_]*`)"
const updateBinding = updateIdentifier + `\s*=\s*\?`

var updatePattern = regexp.MustCompile(`(?i)^UPDATE\s+(` + updateIdentifier + `)\s+SET\s+(` + updateBinding + `(?:\s*,\s*` + updateBinding + `)*)\s+WHERE\s+(` + updateBinding + `(?:\s+AND\s+` + updateBinding + `)*)\s*;?$`)
var updateColumnPattern = regexp.MustCompile(`(` + updateIdentifier + `)\s*=\s*\?`)

type updateColumn struct {
	Column string
	Field  string
}

func (g *Generator) generateBulkUpdateFunction(query *plugin.Query) (*plugin.File, BulkFunction, error) {
	var signature BulkFunction
	entity, ok := strings.CutPrefix(query.GetName(), "Update")
	if !ok || entity == "" || query.GetCmd() != ":execrows" {
		return nil, signature, fmt.Errorf("requires an Update-prefixed :execrows query")
	}
	parts := updatePattern.FindStringSubmatch(NewSQLParser().cleanSQL(query.GetText()))
	if parts == nil {
		return nil, signature, fmt.Errorf("requires UPDATE table SET column = ? WHERE column = ? with only comma-separated assignments and AND predicates")
	}
	assignmentsRaw := updateColumnPattern.FindAllStringSubmatch(parts[2], -1)
	predicatesRaw := updateColumnPattern.FindAllStringSubmatch(parts[3], -1)
	columnsRaw := append(assignmentsRaw, predicatesRaw...)
	if len(columnsRaw) != len(query.GetParams()) {
		return nil, signature, fmt.Errorf("parameter count does not match column bindings")
	}
	columns := make([]updateColumn, 0, len(columnsRaw))
	fields := make(map[string]bool, len(columnsRaw))
	for i, column := range columnsRaw {
		parameter := query.GetParams()[i]
		if int(parameter.GetNumber()) != i+1 || parameter.GetColumn().GetName() == "" {
			return nil, signature, fmt.Errorf("missing parameter metadata for column %s", column[1])
		}
		field := ToCamelCase(parameter.GetColumn().GetName())
		if fields[field] {
			return nil, signature, fmt.Errorf("duplicate parameter field %s: use distinct sqlc.arg names", field)
		}
		fields[field] = true
		columns = append(columns, updateColumn{
			Column: column[1],
			Field:  field,
		})
	}
	assignments, predicates := columns[:len(assignmentsRaw)], columns[len(assignmentsRaw):]
	conditions := make([]string, 0, len(predicates))
	for _, predicate := range predicates {
		for _, assignment := range assignments {
			if strings.EqualFold(strings.Trim(predicate.Column, "`"), strings.Trim(assignment.Column, "`")) {
				return nil, signature, fmt.Errorf("cannot update predicate column %s: MySQL evaluates assignments left to right", predicate.Column)
			}
		}
		conditions = append(conditions, predicate.Column+" = ?")
	}
	signature = BulkFunction{
		Name:         "Update" + pluralize(entity),
		ParamsStruct: query.GetName() + "Params",
		ReturnsRows:  true,
	}
	data := struct {
		Options
		BulkFunction
		Table       string
		Assignments []updateColumn
		Predicates  []updateColumn
		Condition   string
		ArgsPerRow  int
	}{
		Options:      g.options,
		BulkFunction: signature,
		Table:        parts[1],
		Assignments:  assignments,
		Predicates:   predicates,
		Condition:    strings.Join(conditions, " AND "),
		ArgsPerRow:   len(assignments)*(len(predicates)+1) + len(predicates),
	}
	tmpl, err := template.New("bulk_update").Funcs(template.FuncMap{"quote": strconv.Quote}).Parse(bulkUpdateTemplate)
	if err != nil {
		return nil, signature, err
	}
	var content bytes.Buffer
	if err := tmpl.Execute(&content, data); err != nil {
		return nil, signature, err
	}
	formatted, err := format.Source(content.Bytes())
	if err != nil {
		return nil, signature, err
	}
	return &plugin.File{
		Name:     "bulk_" + query.GetFilename() + "_generated.go",
		Contents: formatted,
	}, signature, nil
}
