package validation

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Operation represents an OpenAPI operation with its request schema
type Operation struct {
	Method             string
	Path               string
	OperationID        string
	RequestSchema      map[string]any // JSON Schema for request body
	RequiresBearerAuth bool           // Whether the operation requires Bearer auth
}

// SpecParser parses an OpenAPI specification and extracts operations and schemas
type SpecParser struct {
	operations map[string]*Operation // "POST /v2/keys.setRoles" -> Operation
	schemas    map[string]any        // component schemas for $ref resolution
}

// openAPISpec represents the structure of an OpenAPI 3.1 document
type openAPISpec struct {
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
	Paths map[string]map[string]any `yaml:"paths"`
}

// NewSpecParser creates a new parser from OpenAPI spec bytes
func NewSpecParser(specBytes []byte) (*SpecParser, error) {
	var spec openAPISpec
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	parser := &SpecParser{
		operations: make(map[string]*Operation),
		schemas:    spec.Components.Schemas,
	}

	// Extract operations from paths
	for path, pathItem := range spec.Paths {
		for method, opData := range pathItem {
			// Skip non-operation keys like parameters, servers, etc.
			method = strings.ToUpper(method)
			if method != "GET" && method != "POST" && method != "PUT" &&
				method != "DELETE" && method != "PATCH" && method != "HEAD" && method != "OPTIONS" {
				continue
			}

			op, err := parser.parseOperation(method, path, opData)
			if err != nil {
				return nil, fmt.Errorf("failed to parse operation %s %s: %w", method, path, err)
			}
			if op != nil {
				key := method + " " + path
				parser.operations[key] = op
			}
		}
	}

	return parser, nil
}

// parseOperation extracts operation details from raw YAML data
func (p *SpecParser) parseOperation(method, path string, opData any) (*Operation, error) {
	opMap, ok := opData.(map[string]any)
	if !ok {
		return nil, nil
	}

	// Extract operationId
	operationID := ""
	if opID, ok := opMap["operationId"].(string); ok {
		operationID = opID
	} else {
		// Generate operationId from method and path if not present
		operationID = method + "_" + strings.ReplaceAll(strings.ReplaceAll(path, "/", "_"), ".", "_")
	}

	op := &Operation{
		Method:             method,
		Path:               path,
		OperationID:        operationID,
		RequestSchema:      nil,
		RequiresBearerAuth: requiresBearerAuth(opMap),
	}

	// Extract request body schema
	reqBody, ok := opMap["requestBody"].(map[string]any)
	if !ok {
		return op, nil // No request body
	}

	content, ok := reqBody["content"].(map[string]any)
	if !ok {
		return op, nil
	}

	jsonContent, ok := content["application/json"].(map[string]any)
	if !ok {
		return op, nil
	}

	schema, ok := jsonContent["schema"].(map[string]any)
	if !ok {
		return op, nil
	}

	// Resolve $ref if present
	resolvedSchema, err := p.resolveSchema(schema)
	if err != nil {
		return nil, err
	}
	op.RequestSchema = resolvedSchema

	return op, nil
}

// resolveSchema just returns the schema as-is
// The compiler handles $ref resolution
func (p *SpecParser) resolveSchema(schema map[string]any) (map[string]any, error) {
	return schema, nil
}

// Operations returns all parsed operations
func (p *SpecParser) Operations() map[string]*Operation {
	return p.operations
}

// Schemas returns all component schemas
func (p *SpecParser) Schemas() map[string]any {
	return p.schemas
}

// GetSchemaJSON returns a schema as JSON bytes
func (p *SpecParser) GetSchemaJSON(name string) ([]byte, error) {
	schema, ok := p.schemas[name]
	if !ok {
		return nil, fmt.Errorf("schema not found: %s", name)
	}
	return json.Marshal(schema)
}

// GetFullSpecAsJSON returns the full spec with schemas as JSON for the compiler
func (p *SpecParser) GetFullSpecAsJSON(specBytes []byte) ([]byte, error) {
	var spec map[string]any
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, err
	}
	return json.Marshal(spec)
}

// requiresBearerAuth checks if an operation requires bearer authentication
// by examining the security requirements
func requiresBearerAuth(opMap map[string]any) bool {
	// Check operation-level security
	security, ok := opMap["security"].([]any)
	if !ok {
		// No security defined at operation level - assume it requires auth
		// (global security would apply, which typically requires rootKey)
		return true
	}

	// Empty security array means no auth required
	if len(security) == 0 {
		return false
	}

	// Check if any security requirement uses rootKey (bearer auth)
	for _, req := range security {
		reqMap, ok := req.(map[string]any)
		if !ok {
			continue
		}
		// If rootKey is in the requirements, bearer auth is required
		if _, hasRootKey := reqMap["rootKey"]; hasRootKey {
			return true
		}
	}

	return false
}
