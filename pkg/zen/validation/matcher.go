package validation

import (
	"regexp"
	"strings"
)

// PathMatcher matches HTTP requests to OpenAPI operations
type PathMatcher struct {
	exactPaths    map[string]*Operation // "POST /v2/keys.setRoles" -> Operation (fast exact match)
	templatePaths []templateRoute       // Regex-based template match for paths with parameters
}

// templateRoute holds a compiled regex pattern for path template matching
type templateRoute struct {
	method     string
	pattern    *regexp.Regexp
	paramNames []string
	operation  *Operation
}

// MatchResult holds the matched operation and extracted path parameters
type MatchResult struct {
	Operation  *Operation
	PathParams map[string]string
}

// NewPathMatcher creates a new path matcher from operations
func NewPathMatcher(operations map[string]*Operation) *PathMatcher {
	matcher := &PathMatcher{
		exactPaths:    make(map[string]*Operation),
		templatePaths: nil,
	}

	for key, op := range operations {
		if containsPathParam(op.Path) {
			// Path has parameters like /users/{userId}
			template := compilePathTemplate(op.Method, op.Path, op)
			if template != nil {
				matcher.templatePaths = append(matcher.templatePaths, *template)
			}
		} else {
			// Exact path match
			matcher.exactPaths[key] = op
		}
	}

	return matcher
}

// containsPathParam checks if a path contains path parameters
func containsPathParam(path string) bool {
	return strings.Contains(path, "{") && strings.Contains(path, "}")
}

// compilePathTemplate compiles a path template into a regex pattern
func compilePathTemplate(method, path string, op *Operation) *templateRoute {
	// Convert /users/{userId}/posts/{postId} to regex
	// and extract parameter names

	var paramNames []string
	regexPattern := "^"

	i := 0
	for i < len(path) {
		if path[i] == '{' {
			// Find the closing brace
			end := strings.Index(path[i:], "}")
			if end == -1 {
				// Invalid template
				return nil
			}
			paramName := path[i+1 : i+end]
			paramNames = append(paramNames, paramName)
			// Add a capturing group for the parameter value
			// Match anything except /
			regexPattern += "([^/]+)"
			i = i + end + 1
		} else {
			// Escape special regex characters in literal parts
			char := string(path[i])
			if strings.ContainsAny(char, ".+*?^$()[]{}|\\") {
				regexPattern += "\\" + char
			} else {
				regexPattern += char
			}
			i++
		}
	}
	regexPattern += "$"

	compiled, err := regexp.Compile(regexPattern)
	if err != nil {
		return nil
	}

	return &templateRoute{
		method:     strings.ToUpper(method),
		pattern:    compiled,
		paramNames: paramNames,
		operation:  op,
	}
}

// Match finds the operation for a given HTTP method and path
// Returns a MatchResult with the operation and any extracted path parameters
func (m *PathMatcher) Match(method, path string) (*MatchResult, bool) {
	// Normalize method to uppercase
	method = strings.ToUpper(method)

	// Try exact match first (most API paths are exact)
	key := method + " " + path
	if op, ok := m.exactPaths[key]; ok {
		return &MatchResult{
			Operation:  op,
			PathParams: nil,
		}, true
	}

	// Try template matches
	for _, template := range m.templatePaths {
		if template.method != method {
			continue
		}

		matches := template.pattern.FindStringSubmatch(path)
		if matches == nil {
			continue
		}

		// Extract path parameters
		pathParams := make(map[string]string)
		for i, name := range template.paramNames {
			if i+1 < len(matches) {
				pathParams[name] = matches[i+1]
			}
		}

		return &MatchResult{
			Operation:  template.operation,
			PathParams: pathParams,
		}, true
	}

	return nil, false
}
