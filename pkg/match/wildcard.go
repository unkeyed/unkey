package match

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Wildcard reports whether s matches pattern.
//
// The only wildcard syntax is '*', which matches any sequence of characters
// except newlines, including an empty sequence. Wildcard is intended for
// single-line strings; strings or patterns containing CR or LF are invalid. All
// other characters are matched literally, including regular expression
// metacharacters such as '.', '+', and '?'. Unlike [MatchWatchPaths], Wildcard
// does not treat '/' as a path separator.
//
// Wildcard returns an error when s or pattern contains a newline.
//
// Examples:
//
//	Wildcard("test@gmail.com", "*@gmail.com") returns true.
//	Wildcard("test@yahoo.com", "*@gmail.com") returns false.
//	Wildcard("hello world", "hello*") returns true.
//	Wildcard("hello world", "*world") returns true.
//	Wildcard("hello world", "h*d") returns true.
func Wildcard(s, pattern string) (bool, error) {
	if !isSingleLine(pattern) {
		return false, fault.New("wildcard pattern must be a single-line string",
			fault.Public("Wildcard patterns must be single-line values."),
		)
	}
	if !isSingleLine(s) {
		return false, fault.New("wildcard input must be a single-line string",
			fault.Public("Wildcard matching only supports single-line values."),
		)
	}

	if !strings.Contains(pattern, "*") {
		return s == pattern, nil
	}

	// Split the pattern into the literal pieces that must appear in order.
	// For example, "ab*cd*ef" becomes ["ab", "cd", "ef"]. The '*' characters
	// are the gaps between these literals and may consume any single-line text.
	parts := strings.Split(pattern, "*")
	remaining := s
	partIndex := 0

	// A pattern without a leading '*' is anchored at the beginning, so its first
	// literal must be a real prefix. Once matched, drop it from the remaining
	// input so later literals are searched after it.
	if !strings.HasPrefix(pattern, "*") {
		prefix := parts[0]
		if !strings.HasPrefix(remaining, prefix) {
			return false, nil
		}
		remaining = strings.TrimPrefix(remaining, prefix)
		partIndex = 1
	}

	// A pattern without a trailing '*' is anchored at the end. Keep the final
	// literal for a suffix check after all middle literals have been consumed.
	partEnd := len(parts)
	if !strings.HasSuffix(pattern, "*") {
		partEnd--
	}

	// Match each middle literal in order. Each successful search advances
	// remaining past the literal, which lets the preceding '*' consume everything
	// before it and prevents later literals from matching out of order.
	for ; partIndex < partEnd; partIndex++ {
		part := parts[partIndex]
		if part == "" {
			continue
		}

		index := strings.Index(remaining, part)
		if index < 0 {
			return false, nil
		}
		remaining = remaining[index+len(part):]
	}

	// A trailing '*' can consume the rest of the input, so all required literals
	// have matched already.
	if strings.HasSuffix(pattern, "*") {
		return true, nil
	}

	// Without a trailing '*', the final literal must end the string.
	return strings.HasSuffix(remaining, parts[len(parts)-1]), nil
}

func isSingleLine(s string) bool {
	return !strings.ContainsAny(s, "\r\n")
}
