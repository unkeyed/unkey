package match

import (
	"regexp"
	"strings"
)

// Wildcard checks if a string matches a wildcard pattern.
// The pattern can contain '*' as a wildcard that matches any sequence of characters.
//
// Examples:
//   - Wildcard("test@gmail.com", "*@gmail.com") returns true
//   - Wildcard("test@yahoo.com", "*@gmail.com") returns false
//   - Wildcard("hello world", "hello*") returns true
//   - Wildcard("hello world", "*world") returns true
//   - Wildcard("hello world", "h*d") returns true
func Wildcard(s, pattern string) (bool, error) {
	// Fast path for patterns without wildcards
	if !strings.Contains(pattern, "*") {
		return s == pattern, nil
	}

	// Convert wildcard pattern to regex pattern
	// Escape special regex characters except *
	regexPattern := ""
	for i := 0; i < len(pattern); i++ {
		ch := pattern[i]
		switch ch {
		case '*':
			regexPattern += ".*"
		case '.', '^', '$', '+', '?', '{', '}', '[', ']', '(', ')', '|', '\\':
			regexPattern += "\\" + string(ch)
		default:
			regexPattern += string(ch)
		}
	}

	// Anchor the pattern to match the entire string
	regexPattern = "^" + regexPattern + "$"

	// Check if the pattern matches
	matched, err := regexp.MatchString(regexPattern, s)
	return matched, err
}
