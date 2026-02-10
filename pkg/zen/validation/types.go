package validation

import (
	"encoding/json"
)

// SchemaType represents the type of a JSON Schema as a type-safe enum
type SchemaType int

const (
	SchemaTypeUnknown SchemaType = iota
	SchemaTypeString
	SchemaTypeInteger
	SchemaTypeNumber
	SchemaTypeBoolean
	SchemaTypeArray
	SchemaTypeObject
)

// String returns the string representation of SchemaType
func (s SchemaType) String() string {
	switch s {
	case SchemaTypeUnknown:
		return "string"
	case SchemaTypeString:
		return "string"
	case SchemaTypeInteger:
		return "integer"
	case SchemaTypeNumber:
		return "number"
	case SchemaTypeBoolean:
		return "boolean"
	case SchemaTypeArray:
		return "array"
	case SchemaTypeObject:
		return "object"
	}
	return "string"
}

// ParseSchemaType converts a string type to SchemaType enum
func ParseSchemaType(s string) SchemaType {
	switch s {
	case "string":
		return SchemaTypeString
	case "integer":
		return SchemaTypeInteger
	case "number":
		return SchemaTypeNumber
	case "boolean":
		return SchemaTypeBoolean
	case "array":
		return SchemaTypeArray
	case "object":
		return SchemaTypeObject
	default:
		return SchemaTypeUnknown
	}
}

// ValueKind discriminates the kind of value in CoercedValue
type ValueKind int

const (
	ValueKindNil ValueKind = iota
	ValueKindString
	ValueKindInteger
	ValueKindNumber
	ValueKindBoolean
	ValueKindArray
	ValueKindObject
)

// CoercedValue is a discriminated union representing a coerced parameter value.
// Use the Kind() method to determine which accessor to use.
type CoercedValue struct {
	kind    ValueKind
	str     string
	integer int64
	number  float64
	boolean bool
	array   []CoercedValue
	object  map[string]CoercedValue
}

// NilValue returns a CoercedValue representing nil
func NilValue() CoercedValue {
	return CoercedValue{
		kind:    ValueKindNil,
		str:     "",
		integer: 0,
		number:  0,
		boolean: false,
		array:   nil,
		object:  nil,
	}
}

// StringValue creates a CoercedValue containing a string
func StringValue(s string) CoercedValue {
	return CoercedValue{
		kind:    ValueKindString,
		str:     s,
		integer: 0,
		number:  0,
		boolean: false,
		array:   nil,
		object:  nil,
	}
}

// IntegerValue creates a CoercedValue containing an integer
func IntegerValue(i int64) CoercedValue {
	return CoercedValue{
		kind:    ValueKindInteger,
		str:     "",
		integer: i,
		number:  0,
		boolean: false,
		array:   nil,
		object:  nil,
	}
}

// NumberValue creates a CoercedValue containing a floating-point number
func NumberValue(f float64) CoercedValue {
	return CoercedValue{
		kind:    ValueKindNumber,
		str:     "",
		integer: 0,
		number:  f,
		boolean: false,
		array:   nil,
		object:  nil,
	}
}

// BooleanValue creates a CoercedValue containing a boolean
func BooleanValue(b bool) CoercedValue {
	return CoercedValue{
		kind:    ValueKindBoolean,
		str:     "",
		integer: 0,
		number:  0,
		boolean: b,
		array:   nil,
		object:  nil,
	}
}

// ArrayValue creates a CoercedValue containing an array
func ArrayValue(arr []CoercedValue) CoercedValue {
	return CoercedValue{
		kind:    ValueKindArray,
		str:     "",
		integer: 0,
		number:  0,
		boolean: false,
		array:   arr,
		object:  nil,
	}
}

// ObjectValue creates a CoercedValue containing an object
func ObjectValue(obj map[string]CoercedValue) CoercedValue {
	return CoercedValue{
		kind:    ValueKindObject,
		str:     "",
		integer: 0,
		number:  0,
		boolean: false,
		array:   nil,
		object:  obj,
	}
}

// Kind returns the kind of value stored
func (c CoercedValue) Kind() ValueKind {
	return c.kind
}

// IsNil returns true if this is a nil value
func (c CoercedValue) IsNil() bool {
	return c.kind == ValueKindNil
}

// String returns the string value if this is a string, otherwise ("", false)
func (c CoercedValue) String() (string, bool) {
	if c.kind != ValueKindString {
		return "", false
	}
	return c.str, true
}

// Integer returns the integer value if this is an integer, otherwise (0, false)
func (c CoercedValue) Integer() (int64, bool) {
	if c.kind != ValueKindInteger {
		return 0, false
	}
	return c.integer, true
}

// Number returns the number value if this is a number, otherwise (0, false)
func (c CoercedValue) Number() (float64, bool) {
	if c.kind != ValueKindNumber {
		return 0, false
	}
	return c.number, true
}

// Boolean returns the boolean value if this is a boolean, otherwise (false, false)
func (c CoercedValue) Boolean() (bool, bool) {
	if c.kind != ValueKindBoolean {
		return false, false
	}
	return c.boolean, true
}

// Array returns the array value if this is an array, otherwise (nil, false)
func (c CoercedValue) Array() ([]CoercedValue, bool) {
	if c.kind != ValueKindArray {
		return nil, false
	}
	return c.array, true
}

// Object returns the object value if this is an object, otherwise (nil, false)
func (c CoercedValue) Object() (map[string]CoercedValue, bool) {
	if c.kind != ValueKindObject {
		return nil, false
	}
	return c.object, true
}

// ToAny converts the CoercedValue to any for use with jsonschema.Validate()
func (c CoercedValue) ToAny() any {
	switch c.kind {
	case ValueKindNil:
		return nil
	case ValueKindString:
		return c.str
	case ValueKindInteger:
		return c.integer
	case ValueKindNumber:
		return c.number
	case ValueKindBoolean:
		return c.boolean
	case ValueKindArray:
		result := make([]any, len(c.array))
		for i, v := range c.array {
			result[i] = v.ToAny()
		}
		return result
	case ValueKindObject:
		result := make(map[string]any)
		for k, v := range c.object {
			result[k] = v.ToAny()
		}
		return result
	default:
		return nil
	}
}

// TypedSchema represents a parsed OpenAPI parameter schema with type safety
type TypedSchema struct {
	Type       SchemaType              // The schema type
	Format     string                  // Optional format (e.g., "int64", "date-time")
	Items      *TypedSchema            // For arrays: the schema of array items
	Properties map[string]*TypedSchema // For objects: property schemas
	Raw        json.RawMessage         // Original schema for compilation
}

// ParseTypedSchema parses a schema map into a TypedSchema
func ParseTypedSchema(schema map[string]any) (*TypedSchema, error) {
	if schema == nil {
		return nil, nil
	}

	typed := &TypedSchema{
		Type:       SchemaTypeUnknown,
		Format:     "",
		Items:      nil,
		Properties: nil,
		Raw:        nil,
	}

	// Extract type
	if typeVal, ok := schema["type"]; ok {
		switch t := typeVal.(type) {
		case string:
			typed.Type = ParseSchemaType(t)
		case []any:
			// Handle type array like ["string", "null"] - use first non-null type
			for _, v := range t {
				if str, ok := v.(string); ok && str != "null" {
					typed.Type = ParseSchemaType(str)
					break
				}
			}
		}
	}

	// Extract format
	if format, ok := schema["format"].(string); ok {
		typed.Format = format
	}

	// Extract items for arrays
	if items, ok := schema["items"].(map[string]any); ok {
		itemsTyped, err := ParseTypedSchema(items)
		if err != nil {
			return nil, err
		}
		typed.Items = itemsTyped
	}

	// Extract properties for objects
	if props, ok := schema["properties"].(map[string]any); ok {
		typed.Properties = make(map[string]*TypedSchema)
		for name, propSchema := range props {
			if propMap, ok := propSchema.(map[string]any); ok {
				propTyped, err := ParseTypedSchema(propMap)
				if err != nil {
					return nil, err
				}
				typed.Properties[name] = propTyped
			}
		}
	}

	// Store raw schema for later use
	rawBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	typed.Raw = rawBytes

	return typed, nil
}
