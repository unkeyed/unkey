package streamschema

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
)

var dialects = map[string]bool{
	"https://json-schema.org/draft/2020-12/schema":   true,
	"https://spec.openapis.org/oas/3.1/dialect/base": true,
}

// annotations lists the keywords that carry no assertion and are accepted with
// no effect, each with a check of its value's JSON type. format is here
// because it is an annotation in 2020-12 unless format assertion is enabled.
// Any keyword not handled by [compiler.keyword] and not listed here fails
// compilation, so adding an assertion keyword can never silently drop it.
var annotations = map[string]func(value any) bool{
	"title":        isString,
	"description":  isString,
	"$comment":     isString,
	"format":       isString,
	"default":      isAny,
	"example":      isAny,
	"examples":     isArray,
	"deprecated":   isBool,
	"readOnly":     isBool,
	"writeOnly":    isBool,
	"externalDocs": isObject,
}

type compiler struct {
	nodesMax int
	nodes    int
}

func (c *compiler) schema(value any, location string) (*node, error) {
	if c.nodes >= c.nodesMax {
		return nil, compileError(CompileLimit, location, "schema exceeds SchemaNodesMax")
	}
	c.nodes++

	switch value := value.(type) {
	case bool:
		n := newNode(location)
		n.never = !value
		return n, nil
	case map[string]any:
		n := newNode(location)
		for _, name := range slices.Sorted(maps.Keys(value)) {
			if err := c.keyword(n, name, value[name], location+"/"+escapePointer(name)); err != nil {
				return nil, err
			}
		}
		return n, nil
	default:
		return nil, compileError(CompileInvalid, location, "schema must be an object or a boolean")
	}
}

func (c *compiler) keyword(n *node, name string, value any, location string) error {
	var err error
	switch name {
	case "type":
		n.types, err = parseTypes(value, location)
	case "properties":
		n.properties, err = c.properties(value, location)
	case "required":
		n.required, err = parseRequired(value, location)
	case "additionalProperties":
		n.additional, err = c.schema(value, location)
	case "items":
		if _, tuple := value.([]any); tuple {
			return compileError(CompileUnsupported, location, "the array form of items is not supported")
		}
		n.items, err = c.schema(value, location)
	case "minLength":
		n.lengthMin, err = parseCount(value, location)
	case "maxLength":
		n.lengthMax, err = parseCount(value, location)
	case "minItems":
		n.itemsMin, err = parseCount(value, location)
	case "maxItems":
		n.itemsMax, err = parseCount(value, location)
	case "minimum":
		n.minimum, err = parseBound(value, location, false)
	case "maximum":
		n.maximum, err = parseBound(value, location, false)
	case "exclusiveMinimum":
		n.exclusiveMinimum, err = parseBound(value, location, true)
	case "exclusiveMaximum":
		n.exclusiveMaximum, err = parseBound(value, location, true)
	case "$schema":
		err = checkDialect(value, location, n.location == "")
	default:
		valid, annotation := annotations[name]
		switch {
		case !annotation:
			err = compileError(CompileUnsupported, location, fmt.Sprintf("keyword %q is not supported", name))
		case !valid(value):
			err = compileError(CompileInvalid, location, fmt.Sprintf("keyword %q has a value of the wrong type", name))
		}
	}
	return err
}

func (c *compiler) properties(value any, location string) (map[string]*node, error) {
	members, ok := value.(map[string]any)
	if !ok {
		return nil, compileError(CompileInvalid, location, "properties must be an object")
	}
	properties := make(map[string]*node, len(members))
	for _, name := range slices.Sorted(maps.Keys(members)) {
		child, err := c.schema(members[name], location+"/"+escapePointer(name))
		if err != nil {
			return nil, err
		}
		properties[name] = child
	}
	return properties, nil
}

func parseTypes(value any, location string) (typeSet, error) {
	names, isArray := value.([]any)
	if !isArray {
		names = []any{value}
	} else if len(names) == 0 {
		return 0, compileError(CompileInvalid, location, "type must not be an empty array")
	}

	types := typeSet(0)
	for _, item := range names {
		name, ok := item.(string)
		if !ok {
			return 0, compileError(CompileInvalid, location, "type must be a string or an array of strings")
		}
		bit := typeSet(0)
		for _, entry := range typeNames {
			if entry.name == name {
				bit = entry.bit
			}
		}
		if bit == 0 {
			return 0, compileError(CompileInvalid, location, fmt.Sprintf("unknown type %q", name))
		}
		if types.has(bit) {
			return 0, compileError(CompileInvalid, location, fmt.Sprintf("type %q is listed twice", name))
		}
		types |= bit
	}
	return types, nil
}

func parseRequired(value any, location string) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, compileError(CompileInvalid, location, "required must be an array of strings")
	}
	required := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		name, ok := item.(string)
		if !ok {
			return nil, compileError(CompileInvalid, location, "required must be an array of strings")
		}
		if _, duplicate := seen[name]; duplicate {
			return nil, compileError(CompileInvalid, location, fmt.Sprintf("required lists %q twice", name))
		}
		seen[name] = struct{}{}
		required = append(required, name)
	}
	return required, nil
}

// parseCount parses a non-negative integer keyword value. Integer-valued
// numbers such as 2.0 are accepted, as JSON Schema requires.
func parseCount(value any, location string) (int64, error) {
	d, err := parseNumber(value, location)
	if err != nil {
		return 0, err
	}
	count, ok := d.int64Saturated()
	if !ok {
		return 0, compileError(CompileInvalid, location, "value must be a non-negative integer")
	}
	return count, nil
}

// parseBound parses a numeric bound. The OpenAPI 3.0 boolean form of
// exclusiveMinimum and exclusiveMaximum is reported as unsupported.
func parseBound(value any, location string, exclusive bool) (*decimal, error) {
	if _, isBool := value.(bool); isBool && exclusive {
		return nil, compileError(CompileUnsupported, location, "the boolean form of exclusive bounds from OpenAPI 3.0 is not supported")
	}
	return parseNumber(value, location)
}

func parseNumber(value any, location string) (*decimal, error) {
	number, ok := value.(json.Number)
	if !ok {
		return nil, compileError(CompileInvalid, location, "value must be a number")
	}
	d := &decimal{negative: false, digits: nil, exponent: 0}
	switch parseDecimal([]byte(number), d) {
	case decimalOK:
		return d, nil
	case decimalExponentLimit:
		return nil, compileError(CompileLimit, location, "number exponent exceeds 1e9 in magnitude")
	case decimalSyntax:
		return nil, compileError(CompileInvalid, location, "value must be a number")
	default:
		return nil, compileError(CompileInvalid, location, "value must be a number")
	}
}

func checkDialect(value any, location string, root bool) error {
	uri, ok := value.(string)
	switch {
	case !ok:
		return compileError(CompileInvalid, location, "$schema must be a string")
	case !root:
		return compileError(CompileUnsupported, location, "$schema is only supported at the root")
	case !dialects[uri]:
		return compileError(CompileUnsupported, location, fmt.Sprintf("dialect %q is not supported", uri))
	default:
		return nil
	}
}

func compileError(kind CompileErrorKind, location string, message string) error {
	return &CompileError{Kind: kind, KeywordLocation: location, Message: message, Err: nil}
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isBool(value any) bool {
	_, ok := value.(bool)
	return ok
}

func isArray(value any) bool {
	_, ok := value.([]any)
	return ok
}

func isObject(value any) bool {
	_, ok := value.(map[string]any)
	return ok
}

func isAny(any) bool {
	return true
}
