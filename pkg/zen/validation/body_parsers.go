package validation

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strings"
)

// BodyParser defines the interface for parsing different content types
type BodyParser interface {
	// Parse parses the request body according to the content type
	// Returns the parsed data that can be validated against a JSON schema
	Parse(body []byte, schema map[string]any) (any, error)
	// ContentType returns the MIME type this parser handles
	ContentType() string
}

// FormURLEncodedParser parses application/x-www-form-urlencoded request bodies
type FormURLEncodedParser struct{}

// NewFormURLEncodedParser creates a new FormURLEncodedParser
func NewFormURLEncodedParser() *FormURLEncodedParser {
	return &FormURLEncodedParser{}
}

// ContentType returns the MIME type for form URL-encoded data
func (p *FormURLEncodedParser) ContentType() string {
	return "application/x-www-form-urlencoded"
}

// Parse parses form URL-encoded data and converts it to a map
// The schema is used to determine the expected types for coercion
func (p *FormURLEncodedParser) Parse(body []byte, schema map[string]any) (any, error) {
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return nil, err
	}

	// Extract property schemas for type coercion
	propertySchemas := extractPropertySchemas(schema)

	return formValuesToMap(values, propertySchemas), nil
}

// MultipartFormParser parses multipart/form-data request bodies
type MultipartFormParser struct {
	MaxMemory int64 // Maximum memory to use for file uploads (default: 32MB)
}

// NewMultipartFormParser creates a new MultipartFormParser
func NewMultipartFormParser(maxMemory int64) *MultipartFormParser {
	if maxMemory <= 0 {
		maxMemory = 32 << 20 // 32 MB default
	}
	return &MultipartFormParser{MaxMemory: maxMemory}
}

// ContentType returns the MIME type for multipart form data
func (p *MultipartFormParser) ContentType() string {
	return "multipart/form-data"
}

// Parse parses multipart form data
// Note: This requires access to the original *http.Request, so it returns
// the parsed values and files separately
func (p *MultipartFormParser) Parse(body []byte, schema map[string]any) (any, error) {
	// Multipart parsing is tricky with just bytes - normally we'd use r.ParseMultipartForm
	// For validation purposes, we'll try to parse it from bytes
	// This is a simplified version that may not handle all multipart edge cases
	return p.parseMultipartBytes(body, schema)
}

// ParseMultipartValues parses already-parsed multipart form values into a map
// This is the preferred method when you have access to *http.Request
func (p *MultipartFormParser) ParseMultipartValues(values url.Values, files map[string][]*multipart.FileHeader, schema map[string]any) any {
	propertySchemas := extractPropertySchemas(schema)
	result := formValuesToMap(values, propertySchemas)

	// Add file information to the result
	resultMap, ok := result.(map[string]any)
	if !ok {
		resultMap = make(map[string]any)
	}

	for fieldName, fileHeaders := range files {
		if len(fileHeaders) == 1 {
			// Single file
			resultMap[fieldName] = map[string]any{
				"filename":    fileHeaders[0].Filename,
				"size":        fileHeaders[0].Size,
				"contentType": fileHeaders[0].Header.Get("Content-Type"),
			}
		} else if len(fileHeaders) > 1 {
			// Multiple files
			fileInfos := make([]any, len(fileHeaders))
			for i, fh := range fileHeaders {
				fileInfos[i] = map[string]any{
					"filename":    fh.Filename,
					"size":        fh.Size,
					"contentType": fh.Header.Get("Content-Type"),
				}
			}
			resultMap[fieldName] = fileInfos
		}
	}

	return resultMap
}

// parseMultipartBytes attempts to parse multipart data from raw bytes
// This is a simplified parser for validation purposes
func (p *MultipartFormParser) parseMultipartBytes(body []byte, schema map[string]any) (any, error) {
	// For multipart data, we need the boundary from the Content-Type header
	// Without it, we can't reliably parse multipart data
	return nil, fmt.Errorf("multipart parsing requires boundary; use ParseMultipartRequest instead")
}

// extractPropertySchemas extracts property type information from a JSON schema
func extractPropertySchemas(schema map[string]any) map[string]string {
	result := make(map[string]string)

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return result
	}

	for propName, propSchema := range properties {
		propMap, ok := propSchema.(map[string]any)
		if !ok {
			continue
		}

		// Get the type
		if typeVal, ok := propMap["type"]; ok {
			if typeStr, ok := typeVal.(string); ok {
				result[propName] = typeStr
			} else if typeArr, ok := typeVal.([]any); ok {
				// Type array - find the non-null type
				for _, t := range typeArr {
					if tStr, ok := t.(string); ok && tStr != "null" {
						result[propName] = tStr
						break
					}
				}
			}
		}
	}

	return result
}

// formValuesToMap converts url.Values to a map with proper type coercion
func formValuesToMap(values url.Values, propertySchemas map[string]string) any {
	result := make(map[string]any)

	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}

		schemaType := propertySchemas[key]
		if schemaType == "" {
			schemaType = "string" // Default to string
		}

		if schemaType == "array" {
			// Multiple values as array
			arr := make([]any, len(vals))
			for i, v := range vals {
				arr[i] = coerceValue(v, "string")
			}
			result[key] = arr
		} else if len(vals) == 1 {
			// Single value
			result[key] = coerceValue(vals[0], schemaType)
		} else {
			// Multiple values but not array type - use first value
			result[key] = coerceValue(vals[0], schemaType)
		}
	}

	return result
}

// ParseFormURLEncoded is a convenience function for parsing form URL-encoded data
func ParseFormURLEncoded(body []byte, schema map[string]any) (any, error) {
	parser := NewFormURLEncodedParser()
	return parser.Parse(body, schema)
}

// ParseRequestBody parses a request body based on content type
// Returns the parsed data suitable for JSON schema validation
func ParseRequestBody(contentType string, body []byte, schema map[string]any) (any, error) {
	// Normalize content type (strip parameters)
	mediaType := contentType
	if idx := strings.Index(contentType, ";"); idx != -1 {
		mediaType = strings.TrimSpace(contentType[:idx])
	}
	mediaType = strings.ToLower(mediaType)

	switch mediaType {
	case "application/x-www-form-urlencoded":
		parser := NewFormURLEncodedParser()
		return parser.Parse(body, schema)
	case "multipart/form-data":
		// For multipart, we need the request object for proper parsing
		// Return the raw body as-is for now
		return nil, nil
	default:
		// For JSON and other types, return nil to indicate no special parsing needed
		return nil, nil
	}
}

// ParseMultipartRequest parses a multipart form request
// This requires access to the original request for proper boundary detection
func ParseMultipartRequest(r io.Reader, boundary string, maxMemory int64, schema map[string]any) (any, error) {
	reader := multipart.NewReader(r, boundary)

	// Parse the form
	form, err := reader.ReadForm(maxMemory)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = form.RemoveAll()
	}()

	// Convert to map
	parser := NewMultipartFormParser(maxMemory)
	return parser.ParseMultipartValues(url.Values(form.Value), form.File, schema), nil
}
