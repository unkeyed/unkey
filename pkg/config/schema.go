package config

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Schema generates a JSON Schema (draft 2020-12) from the struct type T
// and returns the schema as indented JSON bytes.
//
// Field names are derived from struct tags in order of precedence: the `json`
// tag is checked first, then `yaml`, and finally the Go field name is used as
// a fallback. A tag value of "-" causes the field to be omitted.
//
// The `config` struct tag maps to JSON Schema keywords:
//   - required      → adds the field to the "required" array
//   - default=V     → "default"
//   - min=N, max=N  → "minimum", "maximum"
//   - minLength=N, maxLength=N → "minLength", "maxLength"
//   - nonempty      → "minLength": 1 for strings, "minItems": 1 for slices
//   - oneof=a|b|c   → "enum"
//
// Nested structs become nested objects with "additionalProperties": false.
// time.Duration fields map to a string type with a description indicating
// Go duration format (e.g. "5s", "1m30s").
//
// The generated schema is useful for editor autocompletion and validation.
// In a YAML config file, add the following modeline for yaml-language-server
// or VS Code JSON schema support:
//
//	# yaml-language-server: $schema=./config.schema.json
//
// A typical go:generate pattern uses a generate_schema.go file to write the
// schema to disk at build time:
//
//	//go:generate go run generate_schema.go
func Schema[T any]() ([]byte, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	schema := buildSchema(t)
	schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"

	return json.MarshalIndent(schema, "", "  ")
}

func buildSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == reflect.TypeOf(time.Duration(0)) {
		return map[string]any{
			"type":        "string",
			"description": "A Go duration string (e.g. 5s, 1m30s)",
		}
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{
			"type":  "array",
			"items": buildSchema(t.Elem()),
		}
	case reflect.Struct:
		return buildStructSchema(t)
	default:
		return map[string]any{}
	}
}

func buildStructSchema(t reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		name := fieldName(sf)
		if name == "" {
			continue
		}

		ft := sf.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}

		prop := buildSchema(ft)
		applyConfigDirectives(prop, sf, ft)

		properties[name] = prop

		tag := sf.Tag.Get("config")
		if tag != "" && hasDirective(parseTag(tag), "required") {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":                 "object",
		"properties":           properties,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func fieldName(sf reflect.StructField) string {
	if jsonTag := sf.Tag.Get("json"); jsonTag != "" {
		name, _, _ := strings.Cut(jsonTag, ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	if yamlTag := sf.Tag.Get("yaml"); yamlTag != "" {
		name, _, _ := strings.Cut(yamlTag, ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	if tomlTag := sf.Tag.Get("toml"); tomlTag != "" {
		name, _, _ := strings.Cut(tomlTag, ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}

	return sf.Name
}

func applyConfigDirectives(prop map[string]any, sf reflect.StructField, ft reflect.Type) {
	tag := sf.Tag.Get("config")
	if tag == "" {
		return
	}

	for _, d := range parseTag(tag) {
		switch d.name {
		case "default":
			if d.value != "" {
				prop["default"] = parseDefaultValue(d.value)
			}
		case "min":
			if n, err := strconv.ParseFloat(d.value, 64); err == nil {
				prop["minimum"] = n
			}
		case "max":
			if n, err := strconv.ParseFloat(d.value, 64); err == nil {
				prop["maximum"] = n
			}
		case "minLength":
			if n, err := strconv.Atoi(d.value); err == nil {
				prop["minLength"] = n
			}
		case "maxLength":
			if n, err := strconv.Atoi(d.value); err == nil {
				prop["maxLength"] = n
			}
		case "nonempty":
			if ft.Kind() == reflect.String {
				prop["minLength"] = 1
			} else if ft.Kind() == reflect.Slice {
				prop["minItems"] = 1
			}
		case "oneof":
			if d.value != "" {
				options := strings.Split(d.value, "|")
				prop["enum"] = options
			}
		}
	}
}

func parseDefaultValue(s string) any {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}
