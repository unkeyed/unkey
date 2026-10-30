// Package guidance provides compact request metadata for interactive API commands.
package guidance

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:generate go run ../../../../tools/cliguidance -spec ../../../../svc/api/openapi/openapi-generated.yaml -resources ../../resources.go -out schemas_generated.json

// Schema describes form input, not a complete JSON Schema validator. Null union
// members become Nullable; Alternatives retain the order of the non-null choices.
type Schema struct {
	Type         string             `json:"type,omitempty"`
	Description  string             `json:"description,omitempty"`
	Nullable     bool               `json:"nullable,omitempty"`
	Secret       bool               `json:"secret,omitempty"`
	Enum         []string           `json:"enum,omitempty"`
	Properties   map[string]*Schema `json:"properties,omitempty"`
	Required     []string           `json:"required,omitempty"`
	Items        *Schema            `json:"items,omitempty"`
	Alternatives []*Schema          `json:"alternatives,omitempty"`
	Additional   *Schema            `json:"additional,omitempty"`
	Minimum      *float64           `json:"minimum,omitempty"`
	Maximum      *float64           `json:"maximum,omitempty"`
	MinLength    int                `json:"minLength,omitempty"`
	MaxLength    int                `json:"maxLength,omitempty"`
	Pattern      string             `json:"pattern,omitempty"`
	MinItems     int                `json:"minItems,omitempty"`
	MaxItems     int                `json:"maxItems,omitempty"`
	Default      json.RawMessage    `json:"default,omitempty"`
}

//go:embed schemas_generated.json
var schemasJSON []byte

var schemas = sync.OnceValue(func() map[string]*Schema {
	var result map[string]*Schema
	if err := json.Unmarshal(schemasJSON, &result); err != nil {
		panic(err)
	}
	return result
})

// Lookup returns shared, read-only metadata keyed by legacy command names such
// as "projects create-project". Commands without a body have an empty object schema.
func Lookup(legacy string) (*Schema, bool) {
	schema, ok := schemas()[legacy]
	return schema, ok
}
