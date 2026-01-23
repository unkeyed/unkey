package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// openAPIKeywords are keywords that are valid in OpenAPI but not in JSON Schema
// and need to be removed or transformed before compilation
var openAPIKeywords = map[string]bool{
	"example":       true, // OpenAPI uses singular 'example'
	"xml":           true, // OpenAPI XML object
	"externalDocs":  true, // OpenAPI external docs
	"discriminator": true, // OpenAPI discriminator (handled separately by OpenAPI)
	"readOnly":      true, // OpenAPI marks as read-only
	"writeOnly":     true, // OpenAPI marks as write-only
	"deprecated":    true, // OpenAPI deprecation flag
}

// SchemaCompiler compiles and caches JSON schemas for validation
type SchemaCompiler struct {
	schemas map[string]*jsonschema.Schema // operationID -> compiled schema
}

// NewSchemaCompiler creates a new schema compiler from a SpecParser
func NewSchemaCompiler(parser *SpecParser, specBytes []byte) (*SchemaCompiler, error) {
	compiler := jsonschema.NewCompiler()

	// Convert the full spec to JSON and clean up OpenAPI-specific keywords
	fullSpecJSON, err := parser.GetFullSpecAsJSON(specBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert spec to JSON: %w", err)
	}

	// Parse, clean, and re-marshal the full spec
	var fullSpec any
	if err := json.Unmarshal(fullSpecJSON, &fullSpec); err != nil {
		return nil, fmt.Errorf("failed to parse full spec: %w", err)
	}

	cleanedSpec := cleanForJSONSchema(fullSpec)
	cleanedJSON, err := json.Marshal(cleanedSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal cleaned spec: %w", err)
	}

	fullSpecDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(cleanedJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal full spec: %w", err)
	}

	// Register the full spec as the base document
	// All $refs like "#/components/schemas/Foo" will resolve against this
	if err := compiler.AddResource("file:///openapi.json", fullSpecDoc); err != nil {
		return nil, fmt.Errorf("failed to add spec resource: %w", err)
	}

	// Compile schemas for each operation
	schemas := make(map[string]*jsonschema.Schema)

	for _, op := range parser.Operations() {
		if op.RequestSchema == nil {
			continue
		}

		compiledSchema, err := compileOperationSchema(compiler, op)
		if err != nil {
			return nil, fmt.Errorf("failed to compile schema for %s: %w", op.OperationID, err)
		}

		schemas[op.OperationID] = compiledSchema
	}

	return &SchemaCompiler{schemas: schemas}, nil
}

// compileOperationSchema compiles the request schema for an operation
func compileOperationSchema(compiler *jsonschema.Compiler, op *Operation) (*jsonschema.Schema, error) {
	// If the schema has a $ref to a component schema, compile from the full spec
	if ref, ok := op.RequestSchema["$ref"].(string); ok {
		// Transform "#/components/schemas/Name" to "file:///openapi.json#/components/schemas/Name"
		if strings.HasPrefix(ref, "#/") {
			resourceURL := "file:///openapi.json" + ref
			return compiler.Compile(resourceURL)
		}
	}

	// For inline schemas, we need to:
	// 1. Transform any $refs to be absolute
	// 2. Clean OpenAPI-specific keywords
	transformedSchema := transformRefs(op.RequestSchema)

	// Clean up OpenAPI-specific keywords that aren't valid JSON Schema
	cleanedAny := cleanForJSONSchema(transformedSchema)
	cleanedSchema, ok := cleanedAny.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("cleaned schema is not a map")
	}

	schemaJSON, err := json.Marshal(cleanedSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request schema: %w", err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(schemaJSON)))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request schema: %w", err)
	}

	resourceURL := "file:///operations/" + op.OperationID
	if err := compiler.AddResource(resourceURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add operation schema: %w", err)
	}

	return compiler.Compile(resourceURL)
}

// transformRefs recursively transforms all $ref values to be absolute
func transformRefs(schema map[string]any) map[string]any {
	result := make(map[string]any)

	for key, value := range schema {
		if key == "$ref" {
			if ref, ok := value.(string); ok && strings.HasPrefix(ref, "#/") {
				result[key] = "file:///openapi.json" + ref
				continue
			}
		}

		switch v := value.(type) {
		case map[string]any:
			result[key] = transformRefs(v)
		case []any:
			result[key] = transformRefsArray(v)
		default:
			result[key] = value
		}
	}

	return result
}

// transformRefsArray handles arrays in the schema
func transformRefsArray(arr []any) []any {
	result := make([]any, len(arr))

	for i, item := range arr {
		if m, ok := item.(map[string]any); ok {
			result[i] = transformRefs(m)
		} else {
			result[i] = item
		}
	}

	return result
}

// Get returns the compiled schema for an operation ID
func (c *SchemaCompiler) Get(operationID string) *jsonschema.Schema {
	return c.schemas[operationID]
}

// Has returns true if a compiled schema exists for the operation ID
func (c *SchemaCompiler) Has(operationID string) bool {
	_, ok := c.schemas[operationID]
	return ok
}

// cleanForJSONSchema removes OpenAPI-specific keywords and transforms structures
// that are valid in OpenAPI but not in strict JSON Schema 2020-12
func cleanForJSONSchema(data any) any {
	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)

		for key, value := range v {
			// Skip OpenAPI-specific keywords that aren't valid JSON Schema
			if openAPIKeywords[key] {
				continue
			}

			// Transform 'examples' from OpenAPI object format to JSON Schema array format
			if key == "examples" {
				if examplesMap, ok := value.(map[string]any); ok {
					// Convert object examples to array (just extract the values)
					arr := make([]any, 0, len(examplesMap))
					for _, ex := range examplesMap {
						if exObj, ok := ex.(map[string]any); ok {
							if val, hasValue := exObj["value"]; hasValue {
								arr = append(arr, val)
							}
						} else {
							arr = append(arr, ex)
						}
					}
					if len(arr) > 0 {
						result[key] = arr
					}
					continue
				}
			}

			// Fix 'type' arrays that contain actual null values instead of the string "null"
			// YAML parses `- null` (unquoted) as nil, but JSON Schema requires the string "null"
			if key == "type" {
				if typeArr, ok := value.([]any); ok {
					fixedArr := make([]any, len(typeArr))
					for i, item := range typeArr {
						if item == nil {
							fixedArr[i] = "null"
						} else {
							fixedArr[i] = item
						}
					}
					result[key] = fixedArr
					continue
				}
			}

			// Recursively clean nested values
			result[key] = cleanForJSONSchema(value)
		}

		return result

	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = cleanForJSONSchema(item)
		}
		return result

	default:
		return v
	}
}
