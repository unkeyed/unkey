// Package discovery describes canonical API commands without executing them.
package discovery

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:generate go run ../../../../tools/cliguidance -mode discovery -spec ../../../../svc/api/openapi/openapi-generated.yaml -resources ../../resources.go -out schemas_generated.json

//go:embed schemas_generated.json
var schemasJSON []byte

// Endpoint identifies the versioned API operation used by the command.
type Endpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// Request retains OpenAPI request objects verbatim. References resolve against
// Components, not against the interactive form metadata.
type Request struct {
	Source            string                                `json:"source"`
	OpenAPI           string                                `json:"openapi"`
	JSONSchemaDialect string                                `json:"jsonSchemaDialect"`
	RequestBody       json.RawMessage                       `json:"requestBody"`
	Components        map[string]map[string]json.RawMessage `json:"components"`
}

type document struct {
	Source            string                     `json:"source"`
	OpenAPI           string                     `json:"openapi"`
	JSONSchemaDialect string                     `json:"jsonSchemaDialect"`
	Endpoints         map[string]endpoint        `json:"endpoints"`
	Components        map[string]json.RawMessage `json:"components"`
}

type endpoint struct {
	Endpoint
	RequestBody json.RawMessage `json:"requestBody"`
	References  []string        `json:"references"`
}

var schemas = sync.OnceValues(func() (*document, error) {
	var doc document
	if err := json.Unmarshal(schemasJSON, &doc); err != nil {
		return nil, fmt.Errorf("decode discovery schemas: %w", err)
	}
	return &doc, nil
})

func lookup(legacy string) (Endpoint, Request, error) {
	doc, err := schemas()
	if err != nil {
		return Endpoint{}, Request{}, err
	}
	endpoint, ok := doc.Endpoints[legacy]
	if !ok {
		return Endpoint{}, Request{}, fmt.Errorf("missing API discovery metadata for %q", legacy)
	}
	request := Request{
		Source: doc.Source, OpenAPI: doc.OpenAPI, JSONSchemaDialect: doc.JSONSchemaDialect,
		RequestBody: endpoint.RequestBody, Components: map[string]map[string]json.RawMessage{},
	}
	for _, ref := range endpoint.References {
		parts := strings.Split(ref, "/")
		if len(parts) != 4 || parts[0] != "#" || parts[1] != "components" {
			return Endpoint{}, Request{}, fmt.Errorf("invalid component reference %q", ref)
		}
		schema, ok := doc.Components[ref]
		if !ok {
			return Endpoint{}, Request{}, fmt.Errorf("missing discovery reference %q", ref)
		}
		group, name := unescape(parts[2]), unescape(parts[3])
		if request.Components[group] == nil {
			request.Components[group] = map[string]json.RawMessage{}
		}
		request.Components[group][name] = schema
	}
	return endpoint.Endpoint, request, nil
}

func (r Request) object(raw json.RawMessage) (map[string]json.RawMessage, error) {
	seen := map[string]bool{}
	for {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		if object["$ref"] == nil {
			return object, nil
		}
		var ref string
		if err := json.Unmarshal(object["$ref"], &ref); err != nil {
			return nil, err
		}
		parts := strings.Split(ref, "/")
		if len(parts) < 4 || parts[0] != "#" || parts[1] != "components" || seen[ref] {
			return nil, fmt.Errorf("unsupported schema projection reference %q", ref)
		}
		seen[ref] = true
		raw = r.Components[unescape(parts[2])][unescape(parts[3])]
		for _, part := range parts[4:] {
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(raw, &nested); err != nil {
				return nil, err
			}
			raw = nested[unescape(part)]
		}
	}
}

func unescape(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~1", "/"), "~0", "~")
}
