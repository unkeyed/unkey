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
	"nullable":      true, // OpenAPI 3.0 only - replaced by type: ["string", "null"] in 3.1
}

// DiscriminatorInfo holds metadata about a discriminator for oneOf/anyOf optimization
type DiscriminatorInfo struct {
	PropertyName string            // The property used for discrimination (e.g., "type")
	Mapping      map[string]string // Discriminator value -> schema ref mapping
}

// CompiledOperation holds compiled schemas for an operation
type CompiledOperation struct {
	BodySchema     *jsonschema.Schema
	BodySchemaName string // Schema name for error messages (e.g., "V2KeysCreateKeyRequestBody")
	BodyRequired   bool
	ContentTypes   []string
	Parameters     CompiledParameterSet
	Discriminator  *DiscriminatorInfo // Optional discriminator for oneOf/anyOf schemas
}

// SchemaRefResolver is a function that resolves a $ref to its schema
type SchemaRefResolver func(ref string) map[string]any

// SchemaCompiler compiles and caches JSON schemas for validation
type SchemaCompiler struct {
	operations map[string]*CompiledOperation // operationID -> compiled operation
	compiler   *jsonschema.Compiler
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
	operations := make(map[string]*CompiledOperation)

	for _, op := range parser.Operations() {
		compiledOp := &CompiledOperation{
			BodySchema:     nil,
			BodySchemaName: op.RequestBodySchemaName,
			BodyRequired:   op.RequestBodyRequired,
			ContentTypes:   op.RequestContentTypes,
			Discriminator:  nil,
			Parameters: CompiledParameterSet{
				Query:  nil,
				Header: nil,
				Cookie: nil,
				Path:   nil,
			},
		}

		// Compile request body schema
		if op.RequestSchema != nil {
			// Extract discriminator info before cleaning (it's stripped during cleaning)
			// Pass the parser's ref resolver to handle $ref schemas
			compiledOp.Discriminator = extractDiscriminator(op.RequestSchema, parser.ResolveSchemaRef)

			bodySchema, err := compileOperationSchema(compiler, op)
			if err != nil {
				return nil, fmt.Errorf("failed to compile body schema for %s: %w", op.OperationID, err)
			}
			compiledOp.BodySchema = bodySchema
		}

		// Compile parameter schemas
		compiledOp.Parameters, err = compileParameterSchemas(compiler, op.OperationID, op.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to compile parameter schemas for %s: %w", op.OperationID, err)
		}

		operations[op.OperationID] = compiledOp
	}

	return &SchemaCompiler{
		operations: operations,
		compiler:   compiler,
	}, nil
}

// compileParameterSchemas compiles schemas for all parameters in a ParameterSet
func compileParameterSchemas(compiler *jsonschema.Compiler, operationID string, params ParameterSet) (CompiledParameterSet, error) {
	result := CompiledParameterSet{
		Query:  []CompiledParameter{},
		Header: []CompiledParameter{},
		Cookie: []CompiledParameter{},
		Path:   []CompiledParameter{},
	}

	paramGroups := []struct {
		source []Parameter
		target *[]CompiledParameter
		name   string
	}{
		{params.Query, &result.Query, "query"},
		{params.Header, &result.Header, "header"},
		{params.Cookie, &result.Cookie, "cookie"},
		{params.Path, &result.Path, "path"},
	}

	for _, group := range paramGroups {
		for _, param := range group.source {
			compiled, err := compileParameterSchema(compiler, operationID, group.name, param)
			if err != nil {
				return result, err
			}
			*group.target = append(*group.target, compiled)
		}
	}

	return result, nil
}

// compileParameterSchema compiles a single parameter's schema
func compileParameterSchema(compiler *jsonschema.Compiler, operationID, location string, param Parameter) (CompiledParameter, error) {
	// Determine style with defaults
	style := param.Style
	if style == "" {
		style = GetDefaultStyle(param.In)
	}

	// Determine explode with defaults
	explode := GetDefaultExplode(style)
	if param.Explode != nil {
		explode = *param.Explode
	}

	compiled := CompiledParameter{
		Name:            param.Name,
		In:              param.In,
		Required:        param.Required,
		Schema:          nil,
		SchemaType:      ParseSchemaType(extractSchemaType(param.Schema)),
		TypedSchema:     param.TypedSchema,
		Style:           style,
		Explode:         explode,
		AllowEmptyValue: param.AllowEmptyValue,
		AllowReserved:   param.AllowReserved,
	}

	if param.Schema == nil {
		return compiled, nil
	}

	// Transform refs to be absolute
	transformedSchema := transformRefs(param.Schema)

	// Clean OpenAPI-specific keywords
	cleanedAny := cleanForJSONSchema(transformedSchema)
	cleanedSchema, ok := cleanedAny.(map[string]any)
	if !ok {
		return compiled, fmt.Errorf("cleaned parameter schema is not a map")
	}

	schemaJSON, err := json.Marshal(cleanedSchema)
	if err != nil {
		return compiled, fmt.Errorf("failed to marshal parameter schema: %w", err)
	}

	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(string(schemaJSON)))
	if err != nil {
		return compiled, fmt.Errorf("failed to unmarshal parameter schema: %w", err)
	}

	resourceURL := fmt.Sprintf("file:///operations/%s/parameters/%s/%s", operationID, location, param.Name)
	if err := compiler.AddResource(resourceURL, schemaDoc); err != nil {
		return compiled, fmt.Errorf("failed to add parameter schema resource: %w", err)
	}

	compiledSchema, err := compiler.Compile(resourceURL)
	if err != nil {
		return compiled, fmt.Errorf("failed to compile parameter schema: %w", err)
	}

	compiled.Schema = compiledSchema
	return compiled, nil
}

// compileOperationSchema compiles the request schema for an operation
func compileOperationSchema(compiler *jsonschema.Compiler, op *Operation) (*jsonschema.Schema, error) {
	// Fast-path: If the schema is exactly a lone $ref with no sibling keys,
	// we can compile directly from the full spec without transformation.
	// Any sibling constraints (description, type, allOf, etc.) require the full
	// transformRefs + cleanForJSONSchema flow to preserve them.
	if len(op.RequestSchema) == 1 {
		if ref, ok := op.RequestSchema["$ref"].(string); ok && strings.HasPrefix(ref, "#/") {
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

// GetOperation returns the full compiled operation
func (c *SchemaCompiler) GetOperation(operationID string) *CompiledOperation {
	return c.operations[operationID]
}

// extractSchemaType extracts the type from a parameter schema for value coercion
func extractSchemaType(schema map[string]any) string {
	if schema == nil {
		return "string"
	}

	typeVal, ok := schema["type"]
	if !ok {
		return "string"
	}

	// Handle single type
	if typeStr, ok := typeVal.(string); ok {
		return typeStr
	}

	// Handle type array (e.g., ["string", "null"])
	if typeArr, ok := typeVal.([]any); ok {
		for _, t := range typeArr {
			if typeStr, ok := t.(string); ok && typeStr != "null" {
				return typeStr
			}
		}
	}

	return "string"
}

// cleanForJSONSchema removes OpenAPI-specific keywords and transforms structures
// that are valid in OpenAPI but not in strict JSON Schema 2020-12
func cleanForJSONSchema(data any) any {
	switch v := data.(type) {
	case map[string]any:
		return cleanSchemaObject(v)
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

// cleanSchemaObject cleans a single schema object (map)
func cleanSchemaObject(v map[string]any) map[string]any {
	result := make(map[string]any)

	// Handle nullable conversion first (OpenAPI 3.0 -> JSON Schema type array)
	if nullableType := convertNullableType(v); nullableType != nil {
		result["type"] = nullableType
	}

	for key, value := range v {
		if openAPIKeywords[key] {
			continue
		}

		// Skip type if already handled via nullable conversion
		if key == "type" && result["type"] != nil {
			continue
		}

		// Transform 'examples' from OpenAPI object format to JSON Schema array format
		if key == "examples" {
			if transformed := transformOpenAPIExamples(value); transformed != nil {
				result[key] = transformed
			}
			continue
		}

		// Fix 'type' arrays containing nil instead of "null" string
		if key == "type" {
			if typeArr, ok := value.([]any); ok {
				result[key] = fixTypeArrayNulls(typeArr)
				continue
			}
		}

		result[key] = cleanForJSONSchema(value)
	}

	return result
}

// convertNullableType converts OpenAPI 3.0 nullable:true to JSON Schema type array.
// Returns the new type array, or nil if no conversion needed.
func convertNullableType(schema map[string]any) []any {
	nullable, ok := schema["nullable"].(bool)
	if !ok || !nullable {
		return nil
	}

	existingType, hasType := schema["type"]
	if !hasType {
		return []any{"null"}
	}

	switch t := existingType.(type) {
	case string:
		return []any{t, "null"}
	case []any:
		if containsNull(t) {
			return fixTypeArrayNulls(t)
		}
		return append(t, "null")
	}

	return nil
}

// containsNull checks if a type array already contains null
func containsNull(typeArr []any) bool {
	for _, tv := range typeArr {
		if tv == "null" || tv == nil {
			return true
		}
	}
	return false
}

// fixTypeArrayNulls replaces nil values with "null" string in type arrays.
// YAML parses `- null` (unquoted) as nil, but JSON Schema requires the string "null".
func fixTypeArrayNulls(typeArr []any) []any {
	result := make([]any, len(typeArr))
	for i, item := range typeArr {
		if item == nil {
			result[i] = "null"
		} else {
			result[i] = item
		}
	}
	return result
}

// transformOpenAPIExamples converts OpenAPI examples object format to JSON Schema array.
// Returns nil if the value is not in OpenAPI object format.
func transformOpenAPIExamples(value any) []any {
	examplesMap, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	arr := make([]any, 0, len(examplesMap))
	for _, ex := range examplesMap {
		if exObj, ok := ex.(map[string]any); ok {
			if val, hasValue := exObj["value"]; hasValue {
				arr = append(arr, val)
				continue
			}
		}
		arr = append(arr, ex)
	}

	if len(arr) == 0 {
		return nil
	}
	return arr
}

// extractDiscriminator extracts discriminator information from a schema that uses oneOf/anyOf.
// If the schema is a $ref, it will be resolved using the provided resolver.
// Returns nil if no discriminator is present.
func extractDiscriminator(schema map[string]any, resolver SchemaRefResolver) *DiscriminatorInfo {
	schema = resolveSchemaRef(schema, resolver)

	discriminator, ok := schema["discriminator"].(map[string]any)
	if !ok {
		return nil
	}

	propertyName, _ := discriminator["propertyName"].(string)
	if propertyName == "" {
		return nil
	}

	mapping := extractExplicitMapping(discriminator)
	if len(mapping) == 0 {
		mapping = inferMappingFromSchema(schema)
	}

	return &DiscriminatorInfo{
		PropertyName: propertyName,
		Mapping:      mapping,
	}
}

// resolveSchemaRef resolves a $ref if the schema is a lone reference
func resolveSchemaRef(schema map[string]any, resolver SchemaRefResolver) map[string]any {
	if resolver == nil || len(schema) != 1 {
		return schema
	}

	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}

	if resolved := resolver(ref); resolved != nil {
		return resolved
	}
	return schema
}

// extractExplicitMapping extracts the explicit mapping from a discriminator object
func extractExplicitMapping(discriminator map[string]any) map[string]string {
	m, ok := discriminator["mapping"].(map[string]any)
	if !ok {
		return nil
	}

	mapping := make(map[string]string)
	for k, v := range m {
		if ref, ok := v.(string); ok {
			mapping[k] = ref
		}
	}
	return mapping
}

// inferMappingFromSchema infers discriminator mapping from oneOf/anyOf $refs.
// Maps schema name to ref (e.g., "Dog" -> "#/components/schemas/Dog")
func inferMappingFromSchema(schema map[string]any) map[string]string {
	mapping := make(map[string]string)

	for _, keyword := range []string{"oneOf", "anyOf"} {
		arr, ok := schema[keyword].([]any)
		if !ok {
			continue
		}

		for _, item := range arr {
			itemSchema, ok := item.(map[string]any)
			if !ok {
				continue
			}

			ref, ok := itemSchema["$ref"].(string)
			if !ok {
				continue
			}

			parts := strings.Split(ref, "/")
			if len(parts) > 0 {
				name := parts[len(parts)-1]
				mapping[name] = ref
			}
		}
	}

	if len(mapping) == 0 {
		return nil
	}
	return mapping
}
