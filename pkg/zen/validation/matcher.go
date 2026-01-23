package validation

import "strings"

// PathMatcher matches HTTP requests to OpenAPI operations
type PathMatcher struct {
	operations map[string]*Operation // "POST /v2/keys.setRoles" -> Operation
}

// NewPathMatcher creates a new path matcher from operations
func NewPathMatcher(operations map[string]*Operation) *PathMatcher {
	return &PathMatcher{operations: operations}
}

// Match finds the operation for a given HTTP method and path
// Returns the operation and true if found, nil and false otherwise
func (m *PathMatcher) Match(method, path string) (*Operation, bool) {
	// Normalize method to uppercase
	method = strings.ToUpper(method)

	// Try exact match first (most API paths are exact)
	key := method + " " + path
	if op, ok := m.operations[key]; ok {
		return op, true
	}

	return nil, false
}
