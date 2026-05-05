package timing

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
)

// HeaderName is the HTTP header name used for timing entries. This custom header
// was chosen over the standard Server-Timing header because it supports a stricter
// format optimized for debugging Unkey's distributed systems.
const HeaderName = "X-Unkey-Timing"

// Entry represents a single timing measurement that can be serialized to the
// [HeaderName] header.
//
// Name follows Prometheus metric naming conventions: it must start with a letter
// or underscore, and may contain letters, digits, underscores, and colons.
//
// Attributes are optional key-value pairs providing context about the measurement.
// Keys follow the same naming rules as Name. Values must be non-empty, contain only
// printable ASCII (0x21-0x7E), and exclude the characters '{', '}', ',', and '='.
//
// Duration must be non-negative. When serialized, it uses the smallest unit that
// avoids sub-unit fractions (e.g., 1500µs becomes "1.5ms", not "1500us").
type Entry struct {
	// Name identifies the operation being measured (e.g., "cache_get", "db_query").
	Name string
	// Attributes provides optional context (e.g., {"status": "miss", "table": "keys"}).
	Attributes map[string]string
	// Duration is the measured time span.
	Duration time.Duration
}

// String formats the entry as an [HeaderName] header value. The output follows
// the format documented in the package overview: name, optional attributes in
// braces, equals sign, and duration with unit suffix. Attributes are sorted
// alphabetically by key for deterministic output.
func (e Entry) String() string {
	return e.format()
}

// Validate checks that the entry conforms to the timing format specification.
// Returns nil if valid, or an error describing the first constraint violation.
//
// Validation rules:
//   - Name must be non-empty and follow Prometheus naming (start with letter/underscore)
//   - Duration must be non-negative
//   - Attribute keys must follow name rules; values must be non-empty printable ASCII
//     excluding '{', '}', ',', '='
func (e Entry) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("entry name is empty")
	}
	if !isNameStart(e.Name[0]) {
		return fmt.Errorf("invalid entry name")
	}
	for i := 1; i < len(e.Name); i++ {
		if !isNameChar(e.Name[i]) {
			return fmt.Errorf("invalid entry name")
		}
	}
	if e.Duration < 0 {
		return fmt.Errorf("duration must be non-negative")
	}
	for key, value := range e.Attributes {
		if key == "" {
			return fmt.Errorf("label key is empty")
		}
		if !isNameStart(key[0]) {
			return fmt.Errorf("invalid label key")
		}
		for i := 1; i < len(key); i++ {
			if !isNameChar(key[i]) {
				return fmt.Errorf("invalid label key")
			}
		}
		if value == "" {
			return fmt.Errorf("label value is empty")
		}
		for i := 0; i < len(value); i++ {
			switch value[i] {
			case ' ', '\t', '\n', '\r':
				return fmt.Errorf("label value contains whitespace")
			case '{', '}', ',', '=':
				return fmt.Errorf("label value contains invalid character")
			default:
				if value[i] < 0x21 || value[i] > 0x7e {
					return fmt.Errorf("label value contains non-printable character")
				}
			}
		}
	}
	return nil
}

func (e Entry) format() string {
	builder := strings.Builder{}
	builder.WriteString(e.Name)
	if len(e.Attributes) > 0 {
		builder.WriteString("{")
		keys := make([]string, 0, len(e.Attributes))
		for key := range e.Attributes {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for i, key := range keys {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(key)
			builder.WriteString("=")
			builder.WriteString(e.Attributes[key])
		}
		builder.WriteString("}")
	}
	builder.WriteString("=")
	builder.WriteString(formatDuration(e.Duration))
	return builder.String()
}

func formatDuration(duration time.Duration) string {
	if duration >= time.Second {
		return formatDurationUnit(duration, time.Second, "s")
	}
	if duration >= time.Millisecond {
		return formatDurationUnit(duration, time.Millisecond, "ms")
	}
	if duration >= time.Microsecond {
		return formatDurationUnit(duration, time.Microsecond, "us")
	}
	return formatDurationUnit(duration, time.Nanosecond, "ns")
}

func formatDurationUnit(duration time.Duration, unit time.Duration, suffix string) string {
	value := float64(duration) / float64(unit)
	return strconv.FormatFloat(value, 'f', -1, 64) + suffix
}

// Record writes a timing entry to the response headers via the [zen.Session]
// associated with the context. If no session is present or the entry fails
// validation, the call is silently ignored. This allows instrumentation code
// to be added unconditionally without error handling.
//
// Use [Write] when you have direct access to an [http.ResponseWriter] instead
// of a context with a session.
func Record(ctx context.Context, entry Entry) {
	if err := entry.Validate(); err != nil {
		return
	}

	session, ok := zen.SessionFromContext(ctx)
	if !ok {
		return
	}

	session.AddHeader(HeaderName, entry.String())
}

// Write adds a timing entry directly to the response headers. If the entry
// fails validation, the call is silently ignored.
//
// Use [Record] when working with a context that contains a [zen.Session],
// which is the common case in request handlers.
func Write(w http.ResponseWriter, entry Entry) {
	if err := entry.Validate(); err != nil {
		return
	}

	w.Header().Add(HeaderName, entry.String())
}

// ParseEntries parses a comma-separated string of timing entries. This is useful
// for parsing concatenated header values when multiple entries are joined in a
// single string. Commas inside attribute braces are treated as attribute separators,
// not entry separators.
//
// Returns an error if the input is empty, contains whitespace, has malformed entries,
// or contains empty entries between commas.
func ParseEntries(input string) ([]Entry, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("header value is empty")
	}
	if hasWhitespace(input) {
		return nil, fmt.Errorf("header value contains whitespace")
	}

	parts, err := splitEntries(input)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(parts))
	for _, part := range parts {
		entry, err := ParseEntry(part)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// ParseEntry parses a single timing entry from its string representation.
// The input must match the format: name[{key=value,...}]=value+unit
//
// Returns the zero Entry and an error if parsing fails. Possible errors include
// empty input, whitespace in input, invalid name or attribute format, missing
// or malformed duration value, and unsupported unit suffix.
//
// Use [ParseEntries] when parsing multiple comma-separated entries.
func ParseEntry(input string) (Entry, error) {
	if input == "" {
		return Entry{}, fmt.Errorf("entry is empty")
	}
	if hasWhitespace(input) {
		return Entry{}, fmt.Errorf("entry contains whitespace")
	}

	name, rest, err := parseName(input)
	if err != nil {
		return Entry{}, err
	}
	if name == "" {
		return Entry{}, fmt.Errorf("entry name is empty")
	}

	attributes := map[string]string{}
	if strings.HasPrefix(rest, "{") {
		labels, remaining, err := parseLabels(rest)
		if err != nil {
			return Entry{}, err
		}
		attributes = labels
		rest = remaining
	}

	if !strings.HasPrefix(rest, "=") {
		return Entry{}, fmt.Errorf("entry missing '='")
	}
	rest = strings.TrimPrefix(rest, "=")
	if rest == "" {
		return Entry{}, fmt.Errorf("entry missing value")
	}

	duration, err := parseDuration(rest)
	if err != nil {
		return Entry{}, err
	}

	return Entry{
		Name:       name,
		Attributes: attributes,
		Duration:   duration,
	}, nil
}

func parseName(input string) (string, string, error) {
	if input == "" {
		return "", "", fmt.Errorf("entry is empty")
	}
	if !isNameStart(input[0]) {
		return "", "", fmt.Errorf("invalid entry name")
	}
	idx := 1
	for idx < len(input) && isNameChar(input[idx]) {
		idx++
	}
	return input[:idx], input[idx:], nil
}

func parseLabels(input string) (map[string]string, string, error) {
	if !strings.HasPrefix(input, "{") {
		return nil, input, fmt.Errorf("labels must start with '{'")
	}
	input = strings.TrimPrefix(input, "{")
	if input == "" {
		return nil, input, fmt.Errorf("labels missing closing '}'")
	}

	labels := map[string]string{}
	for {
		if strings.HasPrefix(input, "}") {
			if len(labels) == 0 {
				return nil, input, fmt.Errorf("labels must not be empty")
			}
			return labels, strings.TrimPrefix(input, "}"), nil
		}

		key, rest, err := parseName(input)
		if err != nil {
			return nil, input, fmt.Errorf("invalid label key: %w", err)
		}
		if key == "" {
			return nil, input, fmt.Errorf("label key is empty")
		}
		if !strings.HasPrefix(rest, "=") {
			return nil, input, fmt.Errorf("label missing '='")
		}
		rest = strings.TrimPrefix(rest, "=")
		value, remaining, err := parseLabelValue(rest)
		if err != nil {
			return nil, input, err
		}
		labels[key] = value
		input = remaining
		if strings.HasPrefix(input, ",") {
			input = strings.TrimPrefix(input, ",")
			if input == "" || strings.HasPrefix(input, "}") {
				return nil, input, fmt.Errorf("label missing after comma")
			}
			continue
		}
	}
}

func parseLabelValue(input string) (string, string, error) {
	end := strings.IndexAny(input, ",}")
	if end == -1 {
		end = len(input)
	}
	value := input[:end]
	if value == "" {
		return "", input, fmt.Errorf("label value is empty")
	}
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ' ', '\t', '\n', '\r':
			return "", input, fmt.Errorf("label value contains whitespace")
		case '{', '}', ',', '=':
			return "", input, fmt.Errorf("label value contains invalid character")
		default:
			if value[i] < 0x21 || value[i] > 0x7e {
				return "", input, fmt.Errorf("label value contains non-printable character")
			}
		}
	}
	return value, input[end:], nil
}

func parseDuration(input string) (time.Duration, error) {
	if input == "" {
		return 0, fmt.Errorf("entry missing value")
	}

	unit, number, err := splitValueUnit(input)
	if err != nil {
		return 0, err
	}
	if err := validateNumber(number); err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid value: %w", err)
	}
	return time.Duration(value * float64(unit)), nil
}

func splitValueUnit(input string) (time.Duration, string, error) {
	units := []struct {
		suffix string
		unit   time.Duration
	}{
		{suffix: "ms", unit: time.Millisecond},
		{suffix: "us", unit: time.Microsecond},
		{suffix: "ns", unit: time.Nanosecond},
		{suffix: "s", unit: time.Second},
	}
	for _, candidate := range units {
		if strings.HasSuffix(input, candidate.suffix) {
			number := strings.TrimSuffix(input, candidate.suffix)
			if number == "" {
				return 0, "", fmt.Errorf("entry missing numeric value")
			}
			return candidate.unit, number, nil
		}
	}

	return 0, "", fmt.Errorf("unsupported unit")
}

func isNameStart(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

func isNameChar(b byte) bool {
	return isNameStart(b) || (b >= '0' && b <= '9') || b == ':'
}

func validateNumber(number string) error {
	if number == "" {
		return fmt.Errorf("entry missing numeric value")
	}
	seenDot := false
	for i := 0; i < len(number); i++ {
		switch number[i] {
		case '.':
			if seenDot {
				return fmt.Errorf("invalid value: multiple decimals")
			}
			if i == 0 || i == len(number)-1 {
				return fmt.Errorf("invalid value: decimal must be between digits")
			}
			seenDot = true
		default:
			if number[i] < '0' || number[i] > '9' {
				return fmt.Errorf("invalid value: contains non-digit")
			}
		}
	}
	return nil
}

func splitEntries(input string) ([]string, error) {
	entries := []string{}
	start := 0
	braceDepth := 0

	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '{':
			braceDepth++
		case '}':
			if braceDepth == 0 {
				return nil, fmt.Errorf("unexpected closing brace")
			}
			braceDepth--
		case ',':
			if braceDepth > 0 {
				continue
			}
			part := input[start:i]
			if part == "" {
				return nil, fmt.Errorf("empty entry")
			}
			entries = append(entries, part)
			start = i + 1
		}
	}

	if braceDepth != 0 {
		return nil, fmt.Errorf("unterminated labels")
	}

	part := input[start:]
	if part == "" {
		return nil, fmt.Errorf("empty entry")
	}
	entries = append(entries, part)

	return entries, nil
}

func hasWhitespace(input string) bool {
	for i := 0; i < len(input); i++ {
		switch input[i] {
		case ' ', '\t', '\n', '\r':
			return true
		}
	}
	return false
}
