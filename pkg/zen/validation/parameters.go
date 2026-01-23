package validation

import (
	"strconv"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// ParameterLocation represents where a parameter is found in the request
type ParameterLocation string

const (
	LocationQuery  ParameterLocation = "query"
	LocationHeader ParameterLocation = "header"
	LocationCookie ParameterLocation = "cookie"
	LocationPath   ParameterLocation = "path"
)

// Parameter represents an OpenAPI parameter definition
type Parameter struct {
	Name            string
	In              ParameterLocation
	Required        bool
	Schema          map[string]any
	Style           string // "form", "simple", "label", "matrix", "spaceDelimited", "pipeDelimited", "deepObject"
	Explode         *bool  // nil = use default for style
	AllowEmptyValue bool   // query params only
	AllowReserved   bool   // query params only
}

// ParameterSet groups parameters by their location
type ParameterSet struct {
	Query  []Parameter
	Header []Parameter
	Cookie []Parameter
	Path   []Parameter
}

// CompiledParameter holds a parameter with its compiled schema
type CompiledParameter struct {
	Name            string
	In              ParameterLocation
	Required        bool
	Schema          *jsonschema.Schema
	SchemaType      string // Original schema type for coercion: "string", "integer", "number", "boolean", "array", "object"
	Style           string // "form", "simple", "label", "matrix", "spaceDelimited", "pipeDelimited", "deepObject"
	Explode         bool   // Whether to explode arrays/objects
	AllowEmptyValue bool   // query params only
	AllowReserved   bool   // query params only
}

// CompiledParameterSet groups compiled parameters by their location
type CompiledParameterSet struct {
	Query  []CompiledParameter
	Header []CompiledParameter
	Cookie []CompiledParameter
	Path   []CompiledParameter
}

// coerceValue attempts to coerce a string value to the appropriate type based on schema type
func coerceValue(value string, schemaType string) any {
	switch schemaType {
	case "integer":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	case "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "boolean":
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	// Default: return as string
	return value
}
