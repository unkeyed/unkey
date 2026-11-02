package main

import (
	"fmt"
	"maps"
	"strings"
)

func compact(document map[string]any, raw any, visiting map[string]bool) (map[string]any, error) {
	if raw == true {
		return map[string]any{"type": "any"}, nil
	}
	source, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unsupported schema %T", raw)
	}
	for _, keyword := range []string{"not", "if", "then", "else", "prefixItems", "patternProperties", "dependentSchemas"} {
		if _, exists := source[keyword]; exists {
			return nil, fmt.Errorf("unsupported schema keyword %s", keyword)
		}
	}
	out := make(map[string]any)
	if ref, ok := source["$ref"].(string); ok {
		if visiting[ref] {
			return nil, fmt.Errorf("recursive reference %q is unsupported", ref)
		}
		resolved, err := resolve(document, ref)
		if err != nil {
			return nil, err
		}
		visiting[ref] = true
		out, err = compact(document, resolved, visiting)
		delete(visiting, ref)
		if err != nil {
			return nil, err
		}
	}
	if all, ok := source["allOf"].([]any); ok {
		if len(all) != 1 {
			return nil, fmt.Errorf("allOf requires exactly one member")
		}
		base, err := compact(document, all[0], visiting)
		if err != nil {
			return nil, err
		}
		maps.Copy(out, base)
	}
	for _, key := range []string{"type", "nullable", "enum", "required", "minimum", "maximum", "minLength", "maxLength", "pattern", "minItems", "maxItems", "default"} {
		if value, exists := source[key]; exists {
			out[key] = value
		}
	}
	if source["x-unkey-redact"] == true || source["writeOnly"] == true {
		out["secret"] = true
	}
	if description, ok := source["description"].(string); ok {
		if description = helper(description); description != "" {
			out["description"] = description
		}
	}
	if properties, ok := source["properties"].(map[string]any); ok {
		children := make(map[string]any, len(properties))
		for name, property := range properties {
			child, err := compact(document, property, visiting)
			if err != nil {
				return nil, fmt.Errorf("property %s: %w", name, err)
			}
			children[name] = child
		}
		if len(children) > 0 {
			out["properties"] = children
		}
		if _, exists := out["type"]; !exists {
			out["type"] = "object"
		}
	}
	if items, exists := source["items"]; exists {
		child, err := compact(document, items, visiting)
		if err != nil {
			return nil, fmt.Errorf("items: %w", err)
		}
		out["items"] = child
	}
	if additional, exists := source["additionalProperties"]; exists && additional != false {
		child, err := compact(document, additional, visiting)
		if err != nil {
			return nil, fmt.Errorf("additional properties: %w", err)
		}
		out["additional"] = child
	}
	if err := choices(document, source, out, visiting); err != nil {
		return nil, err
	}
	if _, exists := out["type"]; !exists {
		switch {
		case out["properties"] != nil, out["required"] != nil, out["additional"] != nil:
			out["type"] = "object"
		case out["items"] != nil:
			out["type"] = "array"
		case out["alternatives"] == nil:
			out["type"] = "any"
		}
	}
	if (source["type"] == "object" || source["properties"] != nil) && out["properties"] == nil && out["required"] == nil && out["alternatives"] == nil && out["additional"] == nil {
		out["additional"] = map[string]any{"type": "any"}
	}
	if source["additionalProperties"] == false {
		delete(out, "additional")
	}
	return out, nil
}

func choices(document, source, out map[string]any, visiting map[string]bool) error {
	var members []any
	if types, ok := out["type"].([]any); ok {
		delete(out, "type")
		for _, kind := range types {
			members = append(members, map[string]any{"type": kind})
		}
	}
	for _, keyword := range []string{"oneOf", "anyOf"} {
		if values, ok := source[keyword].([]any); ok {
			if members != nil {
				return fmt.Errorf("combined type unions, oneOf, or anyOf are unsupported")
			}
			members = values
		}
	}
	if members == nil {
		return nil
	}
	var alternatives []map[string]any
	for _, member := range members {
		child, err := compact(document, member, visiting)
		if err != nil {
			return err
		}
		if child["type"] == "null" {
			out["nullable"] = true
			continue
		}
		alternatives = append(alternatives, child)
	}
	switch len(alternatives) {
	case 0:
		out["type"] = "null"
	case 1:
		base := alternatives[0]
		maps.Copy(base, out)
		maps.Copy(out, base)
	default:
		out["alternatives"] = alternatives
	}
	return nil
}

func helper(description string) string {
	paragraph, _, _ := strings.Cut(strings.TrimSpace(description), "\n\n")
	line := strings.Join(strings.Fields(paragraph), " ")
	if sentence, _, ok := strings.Cut(line, ". "); ok {
		line = sentence + "."
	}
	const lengthMax = 180
	runes := []rune(line)
	if len(runes) <= lengthMax {
		return line
	}
	line = string(runes[:lengthMax-3])
	if space := strings.LastIndexByte(line, ' '); space > 0 {
		line = line[:space]
	}
	return line + "..."
}
