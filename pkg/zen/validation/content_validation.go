package validation

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
)

// ValidateContentEncoding validates string content against the specified encoding type.
// Returns an error if the content does not match the expected encoding.
func ValidateContentEncoding(encoding string, data any) error {
	str, ok := data.(string)
	if !ok {
		return fmt.Errorf("contentEncoding requires string value, got %T", data)
	}

	switch encoding {
	case "base64":
		_, err := base64.StdEncoding.DecodeString(str)
		if err != nil {
			return fmt.Errorf("invalid base64: %w", err)
		}
	case "base64url":
		_, err := base64.URLEncoding.DecodeString(str)
		if err != nil {
			return fmt.Errorf("invalid base64url: %w", err)
		}
	// 7bit, 8bit, binary are for raw transfer, not applicable to JSON
	default:
		// Unknown encoding, skip validation
	}
	return nil
}

// ValidateContentMediaType validates string content against the specified media type.
// Returns an error if the content does not match the expected media type format.
func ValidateContentMediaType(mediaType string, data any) error {
	str, ok := data.(string)
	if !ok {
		return fmt.Errorf("contentMediaType requires string value, got %T", data)
	}

	switch mediaType {
	case "application/json":
		var js any
		if err := json.Unmarshal([]byte(str), &js); err != nil {
			return fmt.Errorf("invalid JSON content: %w", err)
		}
	case "text/xml", "application/xml":
		decoder := xml.NewDecoder(nil)
		decoder = xml.NewDecoder(nil)
		_ = decoder
		// Simple XML validation - try to tokenize the XML
		var xmlData any
		if err := xml.Unmarshal([]byte(str), &xmlData); err != nil {
			return fmt.Errorf("invalid XML content: %w", err)
		}
	// text/plain - no validation needed
	default:
		// Unknown media type, skip validation
	}
	return nil
}

// ContentValidator contains metadata about content validation for a schema property
type ContentValidator struct {
	Path      string // JSON path to the property
	MediaType string // contentMediaType value
	Encoding  string // contentEncoding value
}

// ExtractContentValidators walks a schema and extracts contentMediaType/contentEncoding metadata
func ExtractContentValidators(schema map[string]any, path string) []ContentValidator {
	var validators []ContentValidator

	// Check if this schema has content validation
	mediaType, _ := schema["contentMediaType"].(string)
	encoding, _ := schema["contentEncoding"].(string)

	if mediaType != "" || encoding != "" {
		validators = append(validators, ContentValidator{
			Path:      path,
			MediaType: mediaType,
			Encoding:  encoding,
		})
	}

	// Recursively check properties
	if props, ok := schema["properties"].(map[string]any); ok {
		for propName, propSchema := range props {
			if ps, ok := propSchema.(map[string]any); ok {
				propPath := propName
				if path != "" {
					propPath = path + "." + propName
				}
				validators = append(validators, ExtractContentValidators(ps, propPath)...)
			}
		}
	}

	// Check array items
	if items, ok := schema["items"].(map[string]any); ok {
		itemPath := path + "[]"
		if path == "" {
			itemPath = "[]"
		}
		validators = append(validators, ExtractContentValidators(items, itemPath)...)
	}

	// Check allOf, anyOf, oneOf
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := schema[keyword].([]any); ok {
			for _, item := range arr {
				if itemSchema, ok := item.(map[string]any); ok {
					validators = append(validators, ExtractContentValidators(itemSchema, path)...)
				}
			}
		}
	}

	return validators
}

// GetValueAtPath retrieves a value from data given a JSON path (e.g., "foo.bar" or "items[]")
func GetValueAtPath(data any, path string) (any, bool) {
	if path == "" {
		return data, true
	}

	current := data
	// Simple path parser - handles "foo.bar" and "[]" for arrays
	for path != "" {
		if len(path) >= 2 && path[:2] == "[]" {
			// Array notation - return array items for validation
			path = path[2:]
			if len(path) > 0 && path[0] == '.' {
				path = path[1:]
			}
			// For arrays, we return the whole array for item-by-item validation
			return current, true
		}

		// Find next segment
		dotIdx := -1
		bracketIdx := -1
		for i, c := range path {
			if c == '.' {
				dotIdx = i
				break
			}
			if c == '[' {
				bracketIdx = i
				break
			}
		}

		var segment string
		if dotIdx >= 0 {
			segment = path[:dotIdx]
			path = path[dotIdx+1:]
		} else if bracketIdx >= 0 {
			segment = path[:bracketIdx]
			path = path[bracketIdx:]
		} else {
			segment = path
			path = ""
		}

		if segment == "" {
			continue
		}

		// Navigate to the segment
		if m, ok := current.(map[string]any); ok {
			val, exists := m[segment]
			if !exists {
				return nil, false
			}
			current = val
		} else {
			return nil, false
		}
	}

	return current, true
}
