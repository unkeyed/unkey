package redaction

import (
	"bytes"
)

// stringEnd returns the index just past the string token opening at start,
// reporting false when the string is never closed.
func stringEnd(b []byte, start int) (int, bool) {
	for i := start + 1; i < len(b); {
		rel := bytes.IndexByte(b[i:], '"')
		if rel < 0 {
			return 0, false
		}
		quote := i + rel

		// The quote closes the string unless an odd number of backslashes
		// escapes it.
		backslashes := 0
		for k := quote - 1; k > start && b[k] == '\\'; k-- {
			backslashes++
		}
		if backslashes%2 == 0 {
			return quote + 1, true
		}
		i = quote + 1
	}

	return 0, false
}

// valueEnd returns the index just past the value starting at start. Truncated
// values consume the remainder of the input so nothing sensitive is left behind.
func valueEnd(b []byte, start int) int {
	if start >= len(b) {
		return len(b)
	}

	switch b[start] {
	case '"':
		if end, ok := stringEnd(b, start); ok {
			return end
		}

		return len(b)
	case '{':
		return containerEnd(b, start, '{', '}')
	case '[':
		return containerEnd(b, start, '[', ']')
	default:
		// Number, true, false, null, or garbage: ends at the first structural
		// byte or whitespace.
		for i := start; i < len(b); i++ {
			switch b[i] {
			case ',', '}', ']', ' ', '\t', '\n', '\r':
				return i
			}
		}

		return len(b)
	}
}

// containerEnd returns the index just past the object or array opening at start.
// Depth counts only the outer pair, which is sufficient because any nested
// container of the other type is itself balanced. Braces inside string values
// are skipped.
func containerEnd(b []byte, start int, openByte, closeByte byte) int {
	depth := 0
	for i := start; i < len(b); i++ {
		switch b[i] {
		case '"':
			end, ok := stringEnd(b, i)
			if !ok {
				return len(b)
			}
			i = end - 1
		case openByte:
			depth++
		case closeByte:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return len(b)
}

// isStructural reports whether b can legally follow a complete value.
func isStructural(b byte) bool {
	return b == ',' || b == '}' || b == ']'
}

// skipSpace returns the index of the first byte at or after i that is not JSON
// whitespace.
func skipSpace(b []byte, i int) int {
	for i < len(b) {
		switch b[i] {
		case ' ', '\t', '\n', '\r':
			i++
		default:
			return i
		}
	}

	return i
}
