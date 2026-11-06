// Package tools turns the API's OpenAPI document into MCP tool definitions.
//
// The bundled spec that svc/api embeds is the only input. Every operation
// carries an x-unkey-mcp extension that either lists the toolsets it belongs
// to or says why it is not a tool, so a new endpoint cannot ship without
// someone deciding whether agents get it.
package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Extension is the operation-level spec extension that declares MCP
// membership. It holds exactly one of two keys:
//
//	x-unkey-mcp:
//	  toolsets: [v2, compute]
//
//	x-unkey-mcp:
//	  exclude: "Authenticated by the portal session cookie, which an MCP client cannot hold."
const Extension = "x-unkey-mcp"

// Tool is one API operation exposed as an MCP tool. Every field is derived
// from the spec.
type Tool struct {
	// Name is the public contract users allowlist. It changes only when the
	// operationId changes.
	Name        string
	OperationID string
	// Path is the API route, always called with POST.
	Path        string
	Title       string
	Description string
	// InputSchema is the request body schema with every $ref inlined and
	// example, examples and x-* keywords removed. Its root is always an
	// object schema; operations without a body get {"type":"object"}.
	InputSchema json.RawMessage
	// WrapsBody is true when the request body is not a JSON object, which
	// MCP tool input must be. InputSchema then has one required property,
	// [WrappedBodyProperty], whose value is the request body.
	WrapsBody bool
}

// WrappedBodyProperty holds the request body of a tool whose body is not a
// JSON object, such as the array that ratelimit.multiLimit takes.
const WrappedBodyProperty = "body"

// Registry holds the tools of every toolset. It does not change after [Load]
// and is safe for concurrent use.
type Registry struct {
	byToolset map[ToolsetID][]Tool
}

// Tools returns the tools of one toolset sorted by name. The slice is shared
// and must not be modified.
func (r *Registry) Tools(id ToolsetID) []Tool {
	return r.byToolset[id]
}

// Load builds the registry from a bundled OpenAPI 3.1 document. It returns an
// error naming the operation, never a partial registry, when an operation
// has no valid x-unkey-mcp, a deprecated operation lists toolsets, a tool
// operation is not a parameterless POST with an optional JSON body, a request
// schema is recursive, or a derived name is invalid or collides. svc/mcp calls it once at startup.
func Load(spec []byte) (*Registry, error) {
	var doc specDocument
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return nil, fmt.Errorf("parse spec: %w", err)
	}
	in := inliner{components: doc.Components.Schemas}

	registry := &Registry{byToolset: make(map[ToolsetID][]Tool)}
	operationsByName := make(map[string]string)
	for _, path := range sortedKeys(doc.Paths) {
		for _, method := range sortedKeys(doc.Paths[path]) {
			operation := doc.Paths[path][method]
			toolsets, err := operation.membership()
			if err != nil {
				return nil, fmt.Errorf("%s %s (%s): %w", strings.ToUpper(method), path, operation.OperationID, err)
			}
			if len(toolsets) == 0 {
				continue
			}
			tool, err := operation.tool(method, path, in)
			if err != nil {
				return nil, fmt.Errorf("%s %s (%s): %w", strings.ToUpper(method), path, operation.OperationID, err)
			}
			if other, taken := operationsByName[tool.Name]; taken {
				return nil, fmt.Errorf("operations %s and %s both derive tool name %s", other, tool.OperationID, tool.Name)
			}
			operationsByName[tool.Name] = tool.OperationID
			for _, id := range toolsets {
				registry.byToolset[id] = append(registry.byToolset[id], tool)
			}
		}
	}

	for id := range registry.byToolset {
		slices.SortFunc(registry.byToolset[id], func(a, b Tool) int { return strings.Compare(a.Name, b.Name) })
	}
	return registry, nil
}

type specDocument struct {
	Paths      map[string]map[string]specOperation `yaml:"paths"`
	Components struct {
		Schemas map[string]any `yaml:"schemas"`
	} `yaml:"components"`
}

type specOperation struct {
	OperationID string         `yaml:"operationId"`
	Summary     string         `yaml:"summary"`
	Description string         `yaml:"description"`
	Deprecated  bool           `yaml:"deprecated"`
	Parameters  []any          `yaml:"parameters"`
	RequestBody *specBody      `yaml:"requestBody"`
	MCP         map[string]any `yaml:"x-unkey-mcp"`
}

type specBody struct {
	Content map[string]struct {
		Schema any `yaml:"schema"`
	} `yaml:"content"`
}

// membership returns the toolsets the operation belongs to, or none when it
// is excluded.
func (o specOperation) membership() ([]ToolsetID, error) {
	if o.MCP == nil {
		return nil, fmt.Errorf("missing %s: add toolsets: [...] or exclude: \"<reason>\"", Extension)
	}
	if len(o.MCP) != 1 {
		return nil, fmt.Errorf("%s must have exactly one of toolsets or exclude", Extension)
	}

	if reason, ok := o.MCP["exclude"]; ok {
		text, isString := reason.(string)
		if !isString || strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("%s.exclude must be a non-empty reason", Extension)
		}
		return nil, nil
	}

	raw, ok := o.MCP["toolsets"].([]any)
	if !ok || len(raw) == 0 {
		return nil, fmt.Errorf("%s must have exactly one of toolsets (a non-empty list) or exclude", Extension)
	}
	if o.Deprecated {
		return nil, fmt.Errorf("deprecated operations cannot be tools; use %s.exclude", Extension)
	}
	toolsets := make([]ToolsetID, 0, len(raw))
	for _, value := range raw {
		text, isString := value.(string)
		id := ToolsetID(text)
		if !isString || !knownToolset(id) {
			return nil, fmt.Errorf("unknown toolset %v", value)
		}
		if slices.Contains(toolsets, id) {
			return nil, fmt.Errorf("toolset %s listed twice", id)
		}
		toolsets = append(toolsets, id)
	}
	return toolsets, nil
}

func (o specOperation) tool(method, path string, in inliner) (Tool, error) {
	if !strings.EqualFold(method, http.MethodPost) {
		return Tool{}, fmt.Errorf("tools must be POST operations, got %s", method)
	}
	if len(o.Parameters) > 0 {
		return Tool{}, fmt.Errorf("tools cannot have path, query or header parameters")
	}

	name := toolName(o.OperationID)
	if !nameRule.MatchString(name) {
		return Tool{}, fmt.Errorf("tool name %q does not match %s", name, nameRule)
	}

	schema, wrapsBody, err := o.inputSchema(in)
	if err != nil {
		return Tool{}, err
	}
	return Tool{
		Name:        name,
		OperationID: o.OperationID,
		Path:        path,
		Title:       o.Summary,
		Description: o.Description,
		InputSchema: schema,
		WrapsBody:   wrapsBody,
	}, nil
}

func (o specOperation) inputSchema(in inliner) (json.RawMessage, bool, error) {
	if o.RequestBody == nil {
		return json.RawMessage(`{"type":"object"}`), false, nil
	}
	if len(o.RequestBody.Content) != 1 {
		return nil, false, fmt.Errorf("request body must have exactly one content type, application/json")
	}
	content, ok := o.RequestBody.Content["application/json"]
	if !ok {
		return nil, false, fmt.Errorf("request body must be application/json")
	}

	resolved, err := in.schema(content.Schema, nil)
	if err != nil {
		return nil, false, fmt.Errorf("request schema: %w", err)
	}
	root, isObject := resolved.(map[string]any)
	if !isObject {
		return nil, false, fmt.Errorf("request schema must be a schema object")
	}
	wrapsBody := root["type"] != "object"
	if wrapsBody {
		root = map[string]any{
			"type":                 "object",
			"properties":           map[string]any{WrappedBodyProperty: root},
			"required":             []string{WrappedBodyProperty},
			"additionalProperties": false,
		}
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		return nil, false, fmt.Errorf("encode request schema: %w", err)
	}
	return encoded, wrapsBody, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
