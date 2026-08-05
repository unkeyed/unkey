package validation

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

// Rendering primitives for the parts of a message that come from the
// specification. Nothing here is ever handed a value from the request.

// ratString renders a schema-declared number without the grouping separators and
// trailing zeros that the library's own formatter adds.
func ratString(r *big.Rat) string {
	if r.IsInt() {
		return r.Num().String()
	}

	f, _ := r.Float64()

	return strconv.FormatFloat(f, 'g', -1, 64)
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}

	return strconv.Itoa(n) + " " + plural
}

// specToken guards a string that is supposed to have come from the
// specification. Nothing in this package can prove provenance at runtime, so the
// guard is a length bound: it keeps schema keywords and patterns, and drops
// anything long enough to be a payload if a future library version starts
// putting one there.
func specToken(s string) (string, bool) {
	if s == "" || len(s) > maxSpecTokenLen {
		return "", false
	}

	return s, true
}

const maxSpecTokenLen = 256

func specTokens(values []string) (string, bool) {
	if len(values) == 0 {
		return "", false
	}

	quoted := make([]string, 0, len(values))
	for _, v := range values {
		token, ok := specToken(v)
		if !ok {
			return "", false
		}
		quoted = append(quoted, "'"+token+"'")
	}

	return joinWithAnd(quoted), true
}

// displaySpecValue renders a scalar declared by the schema, such as an enum
// member or a const. Objects and arrays are not rendered: they can be large, and
// a schema is free to declare them inline.
func displaySpecValue(v any) (string, bool) {
	switch value := v.(type) {
	case string:
		token, ok := specToken(value)
		if !ok {
			return "", false
		}

		return "'" + token + "'", true
	case bool:
		return strconv.FormatBool(value), true
	case nil:
		return "null", true
	case float64:
		return strconv.FormatFloat(value, 'g', -1, 64), true
	case int:
		return strconv.Itoa(value), true
	case int64:
		return strconv.FormatInt(value, 10), true
	case json.Number:
		token, ok := specToken(string(value))
		if !ok {
			return "", false
		}

		return token, true
	default:
		return "", false
	}
}

// maxListedValues bounds how many enum members or property names are echoed, so
// a schema with a large enumeration does not turn every rejected request into a
// large response.
const maxListedValues = 12

func displaySpecValues(values []any) (string, bool) {
	if len(values) == 0 {
		return "", false
	}

	rendered := make([]string, 0, len(values))
	for i, v := range values {
		if i == maxListedValues {
			rendered = append(rendered, "…")

			break
		}
		display, ok := displaySpecValue(v)
		if !ok {
			return "", false
		}
		rendered = append(rendered, display)
	}

	return strings.Join(rendered, ", "), true
}

func articled(types []string) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		if _, ok := jsonTypeNames[t]; !ok {
			return nil
		}
		out = append(out, article(t))
	}

	return out
}

// article prefixes a JSON type name so the message reads as English.
func article(jsonType string) string {
	switch jsonType {
	case "integer", "array", "object":
		return "an " + jsonType
	case "null":
		return "null"
	default:
		return "a " + jsonType
	}
}

func joinWithOr(parts []string) string { return joinWith(parts, "or") }

func joinWithAnd(parts []string) string { return joinWith(parts, "and") }

func joinWith(parts []string, conjunction string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " " + conjunction + " " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", " + conjunction + " " + parts[len(parts)-1]
	}
}
