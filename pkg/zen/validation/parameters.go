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
	TypedSchema     *TypedSchema
	Style           string
	Explode         *bool
	AllowEmptyValue bool
	AllowReserved   bool
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
	SchemaType      SchemaType
	TypedSchema     *TypedSchema
	Style           string
	Explode         bool
	AllowEmptyValue bool
	AllowReserved   bool
}

// CompiledParameterSet groups compiled parameters by their location
type CompiledParameterSet struct {
	Query  []CompiledParameter
	Header []CompiledParameter
	Cookie []CompiledParameter
	Path   []CompiledParameter
}

// coerceValue attempts to coerce a string value to the appropriate type based on schema type
func coerceValue(value string, schemaType SchemaType) CoercedValue {
	switch schemaType {
	case SchemaTypeInteger:
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return IntegerValue(i)
		}
	case SchemaTypeNumber:
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return NumberValue(f)
		}
	case SchemaTypeBoolean:
		if b, err := strconv.ParseBool(value); err == nil {
			return BooleanValue(b)
		}
	case SchemaTypeUnknown, SchemaTypeString, SchemaTypeArray, SchemaTypeObject:
		// These types return as string
	}
	return StringValue(value)
}
