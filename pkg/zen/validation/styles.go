package validation

import (
	"net/url"
	"strings"
)

// Schema type constants
const (
	schemaTypeArray  = "array"
	schemaTypeObject = "object"
)

// GetDefaultStyle returns the default style for a parameter location
func GetDefaultStyle(location ParameterLocation) string {
	switch location {
	case LocationQuery, LocationCookie:
		return "form"
	case LocationPath, LocationHeader:
		return "simple"
	default:
		return "form"
	}
}

// GetDefaultExplode returns the default explode value for a style
func GetDefaultExplode(style string) bool {
	// Only "form" style defaults to explode=true
	return style == "form"
}

// ParseByStyle parses parameter values based on the OpenAPI style and explode settings
func ParseByStyle(style string, explode bool, values []string, schemaType string, query url.Values, paramName string) any {
	switch style {
	case "form":
		return parseFormStyle(values, explode, schemaType)
	case "simple":
		if len(values) == 0 {
			return nil
		}
		return parseSimpleStyle(values[0], explode, schemaType)
	case "label":
		if len(values) == 0 {
			return nil
		}
		return parseLabelStyle(values[0], explode, schemaType)
	case "matrix":
		if len(values) == 0 {
			return nil
		}
		return parseMatrixStyle(values[0], explode, schemaType)
	case "spaceDelimited":
		if len(values) == 0 {
			return nil
		}
		return parseSpaceDelimited(values[0], schemaType)
	case "pipeDelimited":
		if len(values) == 0 {
			return nil
		}
		return parsePipeDelimited(values[0], schemaType)
	case "deepObject":
		return parseDeepObject(query, paramName, schemaType)
	default:
		// Default to form style
		return parseFormStyle(values, explode, schemaType)
	}
}

// parseFormStyle parses form-style parameters (default for query and cookie)
// With explode=true: ?id=3&id=4&id=5
// With explode=false: ?id=3,4,5
func parseFormStyle(values []string, explode bool, schemaType string) any {
	if len(values) == 0 {
		return nil
	}

	if schemaType != schemaTypeArray && schemaType != schemaTypeObject {
		// Primitive type - just return the first value coerced
		return coerceValue(values[0], schemaType)
	}

	if schemaType == schemaTypeArray {
		if explode {
			// Each value is a separate array element: ?id=3&id=4&id=5
			result := make([]any, len(values))
			for i, v := range values {
				result[i] = coerceValue(v, "string") // Array items coerced as string by default
			}
			return result
		}
		// Single comma-separated value: ?id=3,4,5
		parts := strings.Split(values[0], ",")
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), "string")
		}
		return result
	}

	// Object type
	if explode {
		// With explode=true, each key=value is separate: ?role=admin&firstName=Alex
		// This requires knowing the object properties, which we can't do here
		// Caller should handle this by passing multiple key-value pairs
		return coerceValue(values[0], schemaType)
	}
	// With explode=false: ?id=role,admin,firstName,Alex
	parts := strings.Split(values[0], ",")
	result := make(map[string]any)
	for i := 0; i+1 < len(parts); i += 2 {
		result[strings.TrimSpace(parts[i])] = coerceValue(strings.TrimSpace(parts[i+1]), "string")
	}
	return result
}

// parseSimpleStyle parses simple-style parameters (default for path and header)
// Arrays: 3,4,5
// Objects (explode=false): role,admin,firstName,Alex
// Objects (explode=true): role=admin,firstName=Alex
func parseSimpleStyle(value string, explode bool, schemaType string) any {
	if value == "" {
		return nil
	}

	if schemaType != schemaTypeArray && schemaType != schemaTypeObject {
		return coerceValue(value, schemaType)
	}

	if schemaType == schemaTypeArray {
		parts := strings.Split(value, ",")
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), "string")
		}
		return result
	}

	// Object type
	if explode {
		// role=admin,firstName=Alex
		parts := strings.Split(value, ",")
		result := make(map[string]any)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				result[strings.TrimSpace(kv[0])] = coerceValue(strings.TrimSpace(kv[1]), "string")
			}
		}
		return result
	}
	// role,admin,firstName,Alex
	parts := strings.Split(value, ",")
	result := make(map[string]any)
	for i := 0; i+1 < len(parts); i += 2 {
		result[strings.TrimSpace(parts[i])] = coerceValue(strings.TrimSpace(parts[i+1]), "string")
	}
	return result
}

// parseLabelStyle parses label-style parameters (.value)
// Arrays: .3.4.5
// Objects (explode=false): .role.admin.firstName.Alex
// Objects (explode=true): .role=admin.firstName=Alex
func parseLabelStyle(value string, explode bool, schemaType string) any {
	// Remove leading dot
	value = strings.TrimPrefix(value, ".")
	if value == "" {
		return nil
	}

	if schemaType != schemaTypeArray && schemaType != schemaTypeObject {
		return coerceValue(value, schemaType)
	}

	if schemaType == schemaTypeArray {
		parts := strings.Split(value, ".")
		result := make([]any, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), "string")
		}
		return result
	}

	// Object type
	if explode {
		// .role=admin.firstName=Alex
		parts := strings.Split(value, ".")
		result := make(map[string]any)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				result[strings.TrimSpace(kv[0])] = coerceValue(strings.TrimSpace(kv[1]), "string")
			}
		}
		return result
	}
	// .role.admin.firstName.Alex
	parts := strings.Split(value, ".")
	result := make(map[string]any)
	for i := 0; i+1 < len(parts); i += 2 {
		result[strings.TrimSpace(parts[i])] = coerceValue(strings.TrimSpace(parts[i+1]), "string")
	}
	return result
}

// parseMatrixStyle parses matrix-style parameters (;name=value)
// Primitive: ;id=5
// Arrays (explode=false): ;id=3,4,5
// Arrays (explode=true): ;id=3;id=4;id=5
// Objects (explode=false): ;id=role,admin,firstName,Alex
// Objects (explode=true): ;role=admin;firstName=Alex
func parseMatrixStyle(value string, explode bool, schemaType string) any {
	// Remove leading semicolon
	value = strings.TrimPrefix(value, ";")
	if value == "" {
		return nil
	}

	if schemaType != schemaTypeArray && schemaType != schemaTypeObject {
		// Primitive: ;id=5 -> parse "id=5" and return value
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 2 {
			return coerceValue(parts[1], schemaType)
		}
		return coerceValue(value, schemaType)
	}

	if schemaType == schemaTypeArray {
		if explode {
			// ;id=3;id=4;id=5
			segments := strings.Split(value, ";")
			result := make([]any, 0, len(segments))
			for _, seg := range segments {
				kv := strings.SplitN(seg, "=", 2)
				if len(kv) == 2 {
					result = append(result, coerceValue(kv[1], "string"))
				}
			}
			return result
		}
		// ;id=3,4,5
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return nil
		}
		items := strings.Split(parts[1], ",")
		result := make([]any, len(items))
		for i, item := range items {
			result[i] = coerceValue(strings.TrimSpace(item), "string")
		}
		return result
	}

	// Object type
	if explode {
		// ;role=admin;firstName=Alex
		segments := strings.Split(value, ";")
		result := make(map[string]any)
		for _, seg := range segments {
			kv := strings.SplitN(seg, "=", 2)
			if len(kv) == 2 {
				result[strings.TrimSpace(kv[0])] = coerceValue(strings.TrimSpace(kv[1]), "string")
			}
		}
		return result
	}
	// ;id=role,admin,firstName,Alex
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return nil
	}
	items := strings.Split(parts[1], ",")
	result := make(map[string]any)
	for i := 0; i+1 < len(items); i += 2 {
		result[strings.TrimSpace(items[i])] = coerceValue(strings.TrimSpace(items[i+1]), "string")
	}
	return result
}

// parseSpaceDelimited parses space-delimited array parameters
// ?id=3%204%205 -> [3, 4, 5]
func parseSpaceDelimited(value string, schemaType string) any {
	if value == "" {
		return nil
	}
	if schemaType != schemaTypeArray {
		return coerceValue(value, schemaType)
	}
	parts := strings.Split(value, " ")
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = coerceValue(strings.TrimSpace(p), "string")
	}
	return result
}

// parsePipeDelimited parses pipe-delimited array parameters
// ?id=3|4|5 -> [3, 4, 5]
func parsePipeDelimited(value string, schemaType string) any {
	if value == "" {
		return nil
	}
	if schemaType != schemaTypeArray {
		return coerceValue(value, schemaType)
	}
	parts := strings.Split(value, "|")
	result := make([]any, len(parts))
	for i, p := range parts {
		result[i] = coerceValue(strings.TrimSpace(p), "string")
	}
	return result
}

// parseDeepObject parses deep object style parameters
// ?filter[name]=foo&filter[age]=30 -> {"name": "foo", "age": 30}
func parseDeepObject(query url.Values, paramName string, schemaType string) any {
	if schemaType != schemaTypeObject {
		return nil
	}

	result := make(map[string]any)
	prefix := paramName + "["

	for key, values := range query {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		// Extract the property name from "filter[name]" -> "name"
		propName := strings.TrimPrefix(key, prefix)
		propName = strings.TrimSuffix(propName, "]")
		if propName != "" && len(values) > 0 {
			result[propName] = coerceValue(values[0], "string")
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}
