package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

type discoveryEndpoint struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"`
	RequestBody any      `json:"requestBody"`
	References  []string `json:"references,omitempty"`
}

func extractDiscovery(data []byte, keys []string) ([]byte, error) {
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse OpenAPI: %w", err)
	}
	version, ok := document["openapi"].(string)
	if !ok || !strings.HasPrefix(version, "3.1.") {
		return nil, fmt.Errorf("discovery requires an OpenAPI 3.1 document, got %q", version)
	}
	dialect, ok := document["jsonSchemaDialect"].(string)
	if !ok {
		dialect = "https://spec.openapis.org/oas/3.1/dialect/base"
	}
	selected, err := selectEndpoints(document, keys)
	if err != nil {
		return nil, err
	}
	endpoints := make(map[string]any, len(selected))
	components := map[string]any{}
	for key, selected := range selected {
		body := selected.Operation["requestBody"]
		references := map[string]any{}
		if err := collectReferences(document, body, references); err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		maps.Copy(components, references)
		endpoints[key] = discoveryEndpoint{
			Method: selected.Method, Path: selected.Path, RequestBody: body,
			References: slices.Sorted(maps.Keys(references)),
		}
	}
	return encodeDiscovery(version, dialect, endpoints, components)
}

func collectReferences(document map[string]any, value any, found map[string]any) error {
	switch value := value.(type) {
	case map[string]any:
		if ref, ok := value["$ref"].(string); ok {
			parts := strings.Split(ref, "/")
			if len(parts) < 4 || parts[0] != "#" || parts[1] != "components" {
				return fmt.Errorf("unsupported reference %q; discovery must be self-contained", ref)
			}
			if _, err := resolve(document, ref); err != nil {
				return err
			}
			component := strings.Join(parts[:4], "/")
			if _, exists := found[component]; !exists {
				resolved, err := resolve(document, component)
				if err != nil {
					return err
				}
				found[component] = resolved
				if err := collectReferences(document, resolved, found); err != nil {
					return err
				}
			}
		}
		for _, child := range value {
			if err := collectReferences(document, child, found); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range value {
			if err := collectReferences(document, child, found); err != nil {
				return err
			}
		}
	}
	return nil
}

func encodeDiscovery(version, dialect string, endpoints, components map[string]any) ([]byte, error) {
	metadata, err := encodeEntries(map[string]any{
		"source": "svc/api/openapi/openapi-generated.yaml", "openapi": version, "jsonSchemaDialect": dialect,
	})
	if err != nil {
		return nil, err
	}
	encoded := bytes.TrimSuffix(metadata, []byte("\n}\n"))
	for _, section := range []struct {
		name    string
		entries map[string]any
	}{{"endpoints", endpoints}, {"components", components}} {
		entries, err := encodeEntries(section.entries)
		if err != nil {
			return nil, err
		}
		name, err := json.Marshal(section.name)
		if err != nil {
			return nil, err
		}
		encoded = append(encoded, ',', '\n', '\t')
		encoded = append(encoded, name...)
		encoded = append(encoded, ':')
		encoded = append(encoded, bytes.TrimSuffix(entries, []byte("\n"))...)
	}
	return append(encoded, '\n', '}', '\n'), nil
}
