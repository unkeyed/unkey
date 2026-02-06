package validation

import (
	"net/url"
	"strings"
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
	return style == "form"
}

// getPropertyType returns the type for a property, defaulting to SchemaTypeString if not found
func getPropertyType(propertyTypes map[string]SchemaType, key string) SchemaType {
	if propertyTypes != nil {
		if t, ok := propertyTypes[key]; ok {
			return t
		}
	}
	return SchemaTypeString
}

// ParseByStyle parses parameter values based on the OpenAPI style and explode settings
// itemType is used for array element coercion, propertyTypes maps property names to their types for objects
func ParseByStyle(style string, explode bool, values []string, schemaType SchemaType, query url.Values, paramName string, itemType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	switch style {
	case "form":
		return parseFormStyle(values, explode, schemaType, itemType, propertyTypes)
	case "simple":
		if len(values) == 0 {
			return NilValue()
		}
		return parseSimpleStyle(values[0], explode, schemaType, itemType, propertyTypes)
	case "label":
		if len(values) == 0 {
			return NilValue()
		}
		return parseLabelStyle(values[0], explode, schemaType, itemType, propertyTypes)
	case "matrix":
		if len(values) == 0 {
			return NilValue()
		}
		return parseMatrixStyle(values[0], explode, schemaType, itemType, propertyTypes)
	case "spaceDelimited":
		if len(values) == 0 {
			return NilValue()
		}
		return parseSpaceDelimited(values[0], schemaType, itemType)
	case "pipeDelimited":
		if len(values) == 0 {
			return NilValue()
		}
		return parsePipeDelimited(values[0], schemaType, itemType)
	case "deepObject":
		return parseDeepObject(query, paramName, schemaType, propertyTypes)
	default:
		return parseFormStyle(values, explode, schemaType, itemType, propertyTypes)
	}
}

// parseFormStyle parses form-style parameters (default for query and cookie)
// With explode=true: ?id=3&id=4&id=5 (arrays) or ?role=admin&firstName=Alex (objects)
// With explode=false: ?id=3,4,5 (arrays) or ?id=role,admin,firstName,Alex (objects)
func parseFormStyle(values []string, explode bool, schemaType SchemaType, itemType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	if len(values) == 0 {
		return NilValue()
	}

	if schemaType != SchemaTypeArray && schemaType != SchemaTypeObject {
		return coerceValue(values[0], schemaType)
	}

	if schemaType == SchemaTypeArray {
		if explode {
			result := make([]CoercedValue, len(values))
			for i, v := range values {
				result[i] = coerceValue(v, itemType)
			}
			return ArrayValue(result)
		}
		parts := strings.Split(values[0], ",")
		result := make([]CoercedValue, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), itemType)
		}
		return ArrayValue(result)
	}

	// Object type
	if explode {
		// Exploded objects: each value is "key=value"
		result := make(map[string]CoercedValue)
		for _, v := range values {
			kv := strings.SplitN(v, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				propType := getPropertyType(propertyTypes, key)
				result[key] = coerceValue(val, propType)
			}
		}
		return ObjectValue(result)
	}
	// Non-exploded objects: key,value,key,value
	parts := strings.Split(values[0], ",")
	result := make(map[string]CoercedValue)
	for i := 0; i+1 < len(parts); i += 2 {
		key := strings.TrimSpace(parts[i])
		val := strings.TrimSpace(parts[i+1])
		propType := getPropertyType(propertyTypes, key)
		result[key] = coerceValue(val, propType)
	}
	return ObjectValue(result)
}

// parseSimpleStyle parses simple-style parameters (default for path and header)
// Arrays: 3,4,5
// Objects (explode=false): role,admin,firstName,Alex
// Objects (explode=true): role=admin,firstName=Alex
func parseSimpleStyle(value string, explode bool, schemaType SchemaType, itemType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	if value == "" {
		return NilValue()
	}

	if schemaType != SchemaTypeArray && schemaType != SchemaTypeObject {
		return coerceValue(value, schemaType)
	}

	if schemaType == SchemaTypeArray {
		parts := strings.Split(value, ",")
		result := make([]CoercedValue, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), itemType)
		}
		return ArrayValue(result)
	}

	if explode {
		parts := strings.Split(value, ",")
		result := make(map[string]CoercedValue)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				propType := getPropertyType(propertyTypes, key)
				result[key] = coerceValue(val, propType)
			}
		}
		return ObjectValue(result)
	}
	parts := strings.Split(value, ",")
	result := make(map[string]CoercedValue)
	for i := 0; i+1 < len(parts); i += 2 {
		key := strings.TrimSpace(parts[i])
		val := strings.TrimSpace(parts[i+1])
		propType := getPropertyType(propertyTypes, key)
		result[key] = coerceValue(val, propType)
	}
	return ObjectValue(result)
}

// parseLabelStyle parses label-style parameters (.value)
// Arrays: .3.4.5
// Objects (explode=false): .role.admin.firstName.Alex
// Objects (explode=true): .role=admin.firstName=Alex
func parseLabelStyle(value string, explode bool, schemaType SchemaType, itemType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	value = strings.TrimPrefix(value, ".")
	if value == "" {
		return NilValue()
	}

	if schemaType != SchemaTypeArray && schemaType != SchemaTypeObject {
		return coerceValue(value, schemaType)
	}

	if schemaType == SchemaTypeArray {
		parts := strings.Split(value, ".")
		result := make([]CoercedValue, len(parts))
		for i, p := range parts {
			result[i] = coerceValue(strings.TrimSpace(p), itemType)
		}
		return ArrayValue(result)
	}

	if explode {
		parts := strings.Split(value, ".")
		result := make(map[string]CoercedValue)
		for _, part := range parts {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				propType := getPropertyType(propertyTypes, key)
				result[key] = coerceValue(val, propType)
			}
		}
		return ObjectValue(result)
	}
	parts := strings.Split(value, ".")
	result := make(map[string]CoercedValue)
	for i := 0; i+1 < len(parts); i += 2 {
		key := strings.TrimSpace(parts[i])
		val := strings.TrimSpace(parts[i+1])
		propType := getPropertyType(propertyTypes, key)
		result[key] = coerceValue(val, propType)
	}
	return ObjectValue(result)
}

// parseMatrixStyle parses matrix-style parameters (;name=value)
// Primitive: ;id=5
// Arrays (explode=false): ;id=3,4,5
// Arrays (explode=true): ;id=3;id=4;id=5
// Objects (explode=false): ;id=role,admin,firstName,Alex
// Objects (explode=true): ;role=admin;firstName=Alex
func parseMatrixStyle(value string, explode bool, schemaType SchemaType, itemType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	value = strings.TrimPrefix(value, ";")
	if value == "" {
		return NilValue()
	}

	if schemaType != SchemaTypeArray && schemaType != SchemaTypeObject {
		parts := strings.SplitN(value, "=", 2)
		if len(parts) == 2 {
			return coerceValue(parts[1], schemaType)
		}
		return coerceValue(value, schemaType)
	}

	if schemaType == SchemaTypeArray {
		if explode {
			segments := strings.Split(value, ";")
			result := make([]CoercedValue, 0, len(segments))
			for _, seg := range segments {
				kv := strings.SplitN(seg, "=", 2)
				if len(kv) == 2 {
					result = append(result, coerceValue(kv[1], itemType))
				}
			}
			return ArrayValue(result)
		}
		parts := strings.SplitN(value, "=", 2)
		if len(parts) != 2 {
			return NilValue()
		}
		items := strings.Split(parts[1], ",")
		result := make([]CoercedValue, len(items))
		for i, item := range items {
			result[i] = coerceValue(strings.TrimSpace(item), itemType)
		}
		return ArrayValue(result)
	}

	if explode {
		segments := strings.Split(value, ";")
		result := make(map[string]CoercedValue)
		for _, seg := range segments {
			kv := strings.SplitN(seg, "=", 2)
			if len(kv) == 2 {
				key := strings.TrimSpace(kv[0])
				val := strings.TrimSpace(kv[1])
				propType := getPropertyType(propertyTypes, key)
				result[key] = coerceValue(val, propType)
			}
		}
		return ObjectValue(result)
	}
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return NilValue()
	}
	items := strings.Split(parts[1], ",")
	result := make(map[string]CoercedValue)
	for i := 0; i+1 < len(items); i += 2 {
		key := strings.TrimSpace(items[i])
		val := strings.TrimSpace(items[i+1])
		propType := getPropertyType(propertyTypes, key)
		result[key] = coerceValue(val, propType)
	}
	return ObjectValue(result)
}

// parseSpaceDelimited parses space-delimited array parameters
// ?id=3%204%205 -> [3, 4, 5]
func parseSpaceDelimited(value string, schemaType SchemaType, itemType SchemaType) CoercedValue {
	if value == "" {
		return NilValue()
	}
	if schemaType != SchemaTypeArray {
		return coerceValue(value, schemaType)
	}
	parts := strings.Split(value, " ")
	result := make([]CoercedValue, len(parts))
	for i, p := range parts {
		result[i] = coerceValue(strings.TrimSpace(p), itemType)
	}
	return ArrayValue(result)
}

// parsePipeDelimited parses pipe-delimited array parameters
// ?id=3|4|5 -> [3, 4, 5]
func parsePipeDelimited(value string, schemaType SchemaType, itemType SchemaType) CoercedValue {
	if value == "" {
		return NilValue()
	}
	if schemaType != SchemaTypeArray {
		return coerceValue(value, schemaType)
	}
	parts := strings.Split(value, "|")
	result := make([]CoercedValue, len(parts))
	for i, p := range parts {
		result[i] = coerceValue(strings.TrimSpace(p), itemType)
	}
	return ArrayValue(result)
}

// parseDeepObject parses deep object style parameters
// ?filter[name]=foo&filter[age]=30 -> {"name": "foo", "age": 30}
func parseDeepObject(query url.Values, paramName string, schemaType SchemaType, propertyTypes map[string]SchemaType) CoercedValue {
	if schemaType != SchemaTypeObject {
		return NilValue()
	}

	result := make(map[string]CoercedValue)
	prefix := paramName + "["

	for key, values := range query {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		propName := strings.TrimPrefix(key, prefix)
		propName = strings.TrimSuffix(propName, "]")
		if propName != "" && len(values) > 0 {
			propType := getPropertyType(propertyTypes, propName)
			if propType == SchemaTypeArray {
				items := make([]CoercedValue, len(values))
				for i, v := range values {
					items[i] = coerceValue(v, SchemaTypeString)
				}
				result[propName] = ArrayValue(items)
			} else {
				result[propName] = coerceValue(values[0], propType)
			}
		}
	}

	if len(result) == 0 {
		return NilValue()
	}
	return ObjectValue(result)
}
