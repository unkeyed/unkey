package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Operation represents an OpenAPI operation with its request schema
type Operation struct {
	Method      string
	Path        string
	OperationID string

	// Request body schema
	RequestSchema map[string]any

	// RequestBodySchemaName is the name of the request body schema (extracted from $ref)
	// e.g., "V2KeysCreateKeyRequestBody"
	RequestBodySchemaName string

	// RequestBodyRequired indicates if a request body is required
	RequestBodyRequired bool

	// RequestContentTypes are the supported content types for the request body
	RequestContentTypes []string

	// Parameters grouped by location
	Parameters ParameterSet

	// Security requirements (empty = no auth required, multiple = OR logic)
	Security []SecurityRequirement
}

// SpecParser parses an OpenAPI specification and extracts operations and schemas
type SpecParser struct {
	operations      map[string]*Operation     // "POST /v2/keys.setRoles" -> Operation
	schemas         map[string]any            // component schemas for $ref resolution
	securitySchemes map[string]SecurityScheme // component security schemes
	globalSecurity  []SecurityRequirement     // default security requirements
}

// openAPISpec represents the structure of an OpenAPI 3.1 document
type openAPISpec struct {
	Components struct {
		Schemas         map[string]any            `yaml:"schemas"`
		SecuritySchemes map[string]map[string]any `yaml:"securitySchemes"`
	} `yaml:"components"`
	Paths    map[string]map[string]any `yaml:"paths"`
	Security []map[string]any          `yaml:"security"`
}

// NewSpecParser creates a new parser from OpenAPI spec bytes
func NewSpecParser(specBytes []byte) (*SpecParser, error) {
	var spec openAPISpec
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	parser := &SpecParser{
		operations:      make(map[string]*Operation),
		schemas:         spec.Components.Schemas,
		securitySchemes: make(map[string]SecurityScheme),
		globalSecurity:  nil,
	}

	// Parse security schemes
	parser.parseSecuritySchemes(spec.Components.SecuritySchemes)

	// Parse global security requirements
	parser.globalSecurity = parser.parseSecurityRequirements(spec.Security)

	// Extract operations from paths
	for path, pathItem := range spec.Paths {
		for method, opData := range pathItem {
			// Skip non-operation keys like parameters, servers, etc.
			method = strings.ToUpper(method)
			if method != "GET" && method != "POST" && method != "PUT" &&
				method != "DELETE" && method != "PATCH" && method != "HEAD" && method != "OPTIONS" {
				continue
			}

			op := parser.parseOperation(method, path, opData)
			if op != nil {
				key := method + " " + path
				parser.operations[key] = op
			}
		}
	}

	return parser, nil
}

// parseSecuritySchemes parses security scheme definitions from components
func (p *SpecParser) parseSecuritySchemes(schemes map[string]map[string]any) {
	for name, schemeData := range schemes {
		schemeType, _ := schemeData["type"].(string)

		scheme := SecurityScheme{
			Type:   SecuritySchemeType(schemeType),
			Scheme: "",
			Name:   "",
			In:     "",
		}

		switch scheme.Type {
		case SecurityTypeHTTP:
			scheme.Scheme, _ = schemeData["scheme"].(string)
		case SecurityTypeAPIKey:
			scheme.Name, _ = schemeData["name"].(string)
			in, _ := schemeData["in"].(string)
			scheme.In = ParameterLocation(in)
		case SecurityTypeOAuth2, SecurityTypeOpenIDConnect:
			// OAuth2 and OpenIDConnect are recognized but don't need additional parsing
			// for presence-only validation
		}

		p.securitySchemes[name] = scheme
	}
}

// parseSecurityRequirements parses a list of security requirements
func (p *SpecParser) parseSecurityRequirements(securityList []map[string]any) []SecurityRequirement {
	if len(securityList) == 0 {
		return nil
	}

	requirements := make([]SecurityRequirement, 0, len(securityList))

	for _, reqData := range securityList {
		req := SecurityRequirement{
			Schemes: make(map[string][]string),
		}

		for schemeName, scopesRaw := range reqData {
			scopes := make([]string, 0)
			if scopesArr, ok := scopesRaw.([]any); ok {
				for _, s := range scopesArr {
					if str, ok := s.(string); ok {
						scopes = append(scopes, str)
					}
				}
			}
			req.Schemes[schemeName] = scopes
		}

		requirements = append(requirements, req)
	}

	return requirements
}

// parseOperation extracts operation details from raw YAML data
func (p *SpecParser) parseOperation(method, path string, opData any) *Operation {
	opMap, ok := opData.(map[string]any)
	if !ok {
		return nil
	}

	// Extract operationId
	operationID, ok := opMap["operationId"].(string)
	if !ok {
		operationID = method + "_" + strings.ReplaceAll(strings.ReplaceAll(path, "/", "_"), ".", "_")
	}

	op := &Operation{
		Method:                method,
		Path:                  path,
		OperationID:           operationID,
		RequestSchema:         nil,
		RequestBodySchemaName: "",
		RequestBodyRequired:   false,
		RequestContentTypes:   nil,
		Parameters: ParameterSet{
			Query:  nil,
			Header: nil,
			Cookie: nil,
			Path:   nil,
		},
		Security: nil,
	}

	// Parse security requirements
	op.Security = p.parseOperationSecurity(opMap)

	// Parse parameters
	op.Parameters = p.parseParameters(opMap)

	// Extract request body schema
	reqBody, ok := opMap["requestBody"].(map[string]any)
	if !ok {
		return op
	}

	// Parse required field
	if required, ok := reqBody["required"].(bool); ok {
		op.RequestBodyRequired = required
	}

	content, ok := reqBody["content"].(map[string]any)
	if !ok {
		return op
	}

	// Extract all supported content types
	for contentType := range content {
		op.RequestContentTypes = append(op.RequestContentTypes, contentType)
	}

	jsonContent, ok := content["application/json"].(map[string]any)
	if !ok {
		return op
	}

	schema, ok := jsonContent["schema"].(map[string]any)
	if !ok {
		return op
	}

	// Extract schema name from $ref if present (e.g., "./V2KeysCreateKeyRequestBody.yaml" -> "V2KeysCreateKeyRequestBody")
	if ref, ok := schema["$ref"].(string); ok {
		op.RequestBodySchemaName = extractSchemaNameFromRef(ref)
	}

	// The compiler handles $ref resolution, so we just store the schema as-is
	op.RequestSchema = schema

	return op
}

// extractSchemaNameFromRef extracts the schema name from a $ref string
// Examples:
//   - "./V2KeysCreateKeyRequestBody.yaml" -> "V2KeysCreateKeyRequestBody"
//   - "#/components/schemas/V2KeysCreateKeyRequestBody" -> "V2KeysCreateKeyRequestBody"
func extractSchemaNameFromRef(ref string) string {
	// Handle local file references like "./V2KeysCreateKeyRequestBody.yaml"
	if strings.HasPrefix(ref, "./") {
		ref = strings.TrimPrefix(ref, "./")
		ref = strings.TrimSuffix(ref, ".yaml")
		ref = strings.TrimSuffix(ref, ".json")
		return ref
	}

	// Handle JSON pointer references like "#/components/schemas/V2KeysCreateKeyRequestBody"
	if strings.HasPrefix(ref, "#/") {
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return ""
}

// parseOperationSecurity parses security requirements for an operation
func (p *SpecParser) parseOperationSecurity(opMap map[string]any) []SecurityRequirement {
	// Check for operation-level security
	securityRaw, hasOperationSecurity := opMap["security"]

	if !hasOperationSecurity {
		// No operation-level security defined, use global security
		return p.globalSecurity
	}

	// Check for explicit empty array (no auth required)
	securityArr, ok := securityRaw.([]any)
	if !ok {
		// Invalid security definition, fall back to global
		return p.globalSecurity
	}

	// Empty array means no auth required
	if len(securityArr) == 0 {
		return []SecurityRequirement{}
	}

	// Convert []any to []map[string]any for parseSecurityRequirements
	securityList := make([]map[string]any, 0, len(securityArr))
	for _, item := range securityArr {
		if m, ok := item.(map[string]any); ok {
			securityList = append(securityList, m)
		}
	}

	return p.parseSecurityRequirements(securityList)
}

// parseParameters parses parameters from an operation
func (p *SpecParser) parseParameters(opMap map[string]any) ParameterSet {
	paramSet := ParameterSet{
		Query:  nil,
		Header: nil,
		Cookie: nil,
		Path:   nil,
	}

	paramsRaw, ok := opMap["parameters"].([]any)
	if !ok {
		return paramSet
	}

	for _, paramRaw := range paramsRaw {
		paramMap, ok := paramRaw.(map[string]any)
		if !ok {
			continue
		}

		param := Parameter{
			Name:            "",
			In:              "",
			Required:        false,
			Schema:          nil,
			TypedSchema:     nil,
			Style:           "",
			Explode:         nil,
			AllowEmptyValue: false,
			AllowReserved:   false,
		}

		if name, ok := paramMap["name"].(string); ok {
			param.Name = name
		}

		if in, ok := paramMap["in"].(string); ok {
			param.In = ParameterLocation(in)
		}

		if required, ok := paramMap["required"].(bool); ok {
			param.Required = required
		}

		if schema, ok := paramMap["schema"].(map[string]any); ok {
			param.Schema = schema
			// Parse typed schema for type-safe handling
			param.TypedSchema, _ = ParseTypedSchema(schema)
		}

		// Parse style (OpenAPI 3.x parameter serialization)
		if style, ok := paramMap["style"].(string); ok {
			param.Style = style
		}

		// Parse explode (defaults depend on style)
		if explode, ok := paramMap["explode"].(bool); ok {
			param.Explode = &explode
		}

		// Parse allowEmptyValue (query params only)
		if allowEmpty, ok := paramMap["allowEmptyValue"].(bool); ok {
			param.AllowEmptyValue = allowEmpty
		}

		// Parse allowReserved (query params only)
		if allowReserved, ok := paramMap["allowReserved"].(bool); ok {
			param.AllowReserved = allowReserved
		}

		// Add to appropriate list based on location
		switch param.In {
		case LocationQuery:
			paramSet.Query = append(paramSet.Query, param)
		case LocationHeader:
			paramSet.Header = append(paramSet.Header, param)
		case LocationCookie:
			paramSet.Cookie = append(paramSet.Cookie, param)
		case LocationPath:
			paramSet.Path = append(paramSet.Path, param)
		}
	}

	return paramSet
}

// Operations returns all parsed operations
func (p *SpecParser) Operations() map[string]*Operation {
	return p.operations
}

// SecuritySchemes returns all security schemes
func (p *SpecParser) SecuritySchemes() map[string]SecurityScheme {
	return p.securitySchemes
}

// GetFullSpecAsJSON returns the full spec with schemas as JSON for the compiler
func (p *SpecParser) GetFullSpecAsJSON(specBytes []byte) ([]byte, error) {
	var spec map[string]any
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}
	return json.Marshal(spec)
}

// ResolveSchemaRef resolves a $ref to its actual schema from components/schemas.
// Returns nil if the ref cannot be resolved.
func (p *SpecParser) ResolveSchemaRef(ref string) map[string]any {
	// Only handle local refs to components/schemas
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return nil
	}

	schemaName := strings.TrimPrefix(ref, prefix)
	if schema, ok := p.schemas[schemaName].(map[string]any); ok {
		return schema
	}
	return nil
}

// Schemas returns the component schemas map
func (p *SpecParser) Schemas() map[string]any {
	return p.schemas
}
