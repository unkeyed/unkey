package tools

import (
	"fmt"
	"strings"
)

const componentRefPrefix = "#/components/schemas/"

// Keywords whose value is a map from a name to a schema. Their keys are
// property or definition names, never keywords, so they are not stripped.
var namedSchemaKeywords = map[string]bool{
	"properties":        true,
	"patternProperties": true,
	"dependentSchemas":  true,
	"$defs":             true,
}

// Keywords whose value is a schema.
var schemaKeywords = map[string]bool{
	"items":                 true,
	"additionalProperties":  true,
	"unevaluatedProperties": true,
	"unevaluatedItems":      true,
	"propertyNames":         true,
	"contains":              true,
	"not":                   true,
	"if":                    true,
	"then":                  true,
	"else":                  true,
}

// Keywords whose value is a list of schemas.
var schemaListKeywords = map[string]bool{
	"allOf":       true,
	"anyOf":       true,
	"oneOf":       true,
	"prefixItems": true,
}

// inliner turns a request body schema into a self-contained JSON Schema:
// every component $ref is replaced by its target, and example, examples and
// x-* keywords are dropped. Examples only repeat what descriptions say, and
// clients that do not resolve references can read the result.
type inliner struct {
	components map[string]any
}

func (in inliner) schema(node any, stack []string) (any, error) {
	object, ok := node.(map[string]any)
	if !ok {
		return node, nil
	}

	if ref, isRef := object["$ref"].(string); isRef {
		return in.ref(ref, object, stack)
	}

	out := make(map[string]any, len(object))
	for keyword, value := range object {
		if stripped(keyword) {
			continue
		}
		resolved, err := in.keyword(keyword, value, stack)
		if err != nil {
			return nil, err
		}
		out[keyword] = resolved
	}
	return out, nil
}

func (in inliner) keyword(keyword string, value any, stack []string) (any, error) {
	switch {
	case schemaKeywords[keyword]:
		return in.schema(value, stack)
	case schemaListKeywords[keyword]:
		list, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("%s must be a list", keyword)
		}
		out := make([]any, len(list))
		for i, item := range list {
			resolved, err := in.schema(item, stack)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	case namedSchemaKeywords[keyword]:
		named, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s must be a map", keyword)
		}
		out := make(map[string]any, len(named))
		for name, item := range named {
			resolved, err := in.schema(item, stack)
			if err != nil {
				return nil, err
			}
			out[name] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}

// ref inlines a component. Keywords next to $ref (OpenAPI 3.1 allows them,
// typically a description) override the component's own.
func (in inliner) ref(ref string, object map[string]any, stack []string) (any, error) {
	name, ok := strings.CutPrefix(ref, componentRefPrefix)
	if !ok {
		return nil, fmt.Errorf("unsupported $ref %q", ref)
	}
	for _, seen := range stack {
		if seen == name {
			return nil, fmt.Errorf("recursive schema %s", strings.Join(append(stack, name), " -> "))
		}
	}
	target, ok := in.components[name]
	if !ok {
		return nil, fmt.Errorf("$ref %q has no component", ref)
	}
	resolved, err := in.schema(target, append(stack, name))
	if err != nil {
		return nil, err
	}

	if len(object) == 1 {
		return resolved, nil
	}
	resolvedObject, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("$ref %q with sibling keywords must point to a schema object", ref)
	}
	merged := make(map[string]any, len(resolvedObject)+len(object))
	for keyword, value := range resolvedObject {
		merged[keyword] = value
	}
	for keyword, value := range object {
		if keyword == "$ref" || stripped(keyword) {
			continue
		}
		override, err := in.keyword(keyword, value, stack)
		if err != nil {
			return nil, err
		}
		merged[keyword] = override
	}
	return merged, nil
}

func stripped(keyword string) bool {
	return keyword == "example" || keyword == "examples" || strings.HasPrefix(keyword, "x-")
}
