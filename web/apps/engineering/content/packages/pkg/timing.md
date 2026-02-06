---
title: timing
description: "defines a small, strict header format for server-side timing"
---

Package timing defines a small, strict header format for server-side timing data and provides helpers to record it through zen sessions.

The package exists because Server-Timing is standardized but too limited for Unkey's debugging needs. We want a header that remains readable in raw HTTP responses, supports optional attributes, and is simple to parse without the complexity of quoted values or browser integration.

### Format

Each entry is encoded as a single header value for the X-Unkey-Timing header. A typical example is:

	X-Unkey-Timing: cache_get{status=miss,cache=ApiByID}=1.25ms

An entry has a name, optional attributes in braces, and a numeric value with a unit suffix. One header value should encode one entry to keep parsing small and avoid quoting edge cases. Whitespace is not permitted anywhere in the entry. Names and label keys follow the same grammar as Prometheus metric names. Label values are unquoted, must be non-empty, and must not contain whitespace or any of the characters '{', '}', ',', or '='. Values are unsigned numbers and cannot use sign or exponent notation.

### Value and Units

Values are integers or floating-point numbers followed by a unit suffix that maps to a [time.Duration](/time#Duration) unit. Supported units are ns, us, ms, and s. The decimal must have digits on both sides if present.

### Examples

Valid entries:

	X-Unkey-Timing: cache_get=1ms
	X-Unkey-Timing: cache_get{status=miss}=1.25ms
	X-Unkey-Timing: db_query{table=keys,status=miss}=53ms

Invalid entries:

	X-Unkey-Timing: cache_get{status=}=1ms
	X-Unkey-Timing: cache_get=.5ms
	X-Unkey-Timing: cache_get=1e3ms
	X-Unkey-Timing: cache_get{status=miss,reason=too slow}=1ms

### Usage

Callers record entries directly on the current request context. If a zen session is present, the entry is serialized and added to the response headers. This keeps observability wiring close to the request without exposing sessions to application code.

	 timing.Record(ctx, timing.Entry{
		Name:     "cache_get",
		Duration: 1500 * time.Microsecond,
		Attributes: map[string]string{
			"status": "miss",
		},
	})

If you need to read these values later, use \[ParseEntry] or \[ParseEntries] on the header values.

### ABNF

The grammar is expressed in ABNF as follows:

entry       = name \[labels] "=" value unit labels      = "{" label \*("," label) "}" label       = name "=" label-value name        = name-start \*(name-char) name-start  = ALPHA / "\_" name-char   = ALPHA / DIGIT / "\_" / ":" value       = 1\*DIGIT \["." 1\*DIGIT] unit        = "ns" / "us" / "ms" / "s" label-value = 1\*(label-char) label-char  = %x21-7E

The label-char set excludes '{', '}', ',', and '=' per the rules above.

### Design Goals

The format is readable in raw HTTP headers, cheap to parse in hot paths, and extensible through optional attributes.

### Non-goals

The format does not aim for browser parsing or Prometheus scrape compatibility.

## Constants

HeaderName is the HTTP header name used for timing entries. This custom header was chosen over the standard Server-Timing header because it supports a stricter format optimized for debugging Unkey's distributed systems.
```go
const HeaderName = "X-Unkey-Timing"
```


## Functions

### func Record

```go
func Record(ctx context.Context, entry Entry)
```

Record writes a timing entry to the response headers via the \[zen.Session] associated with the context. If no session is present or the entry fails validation, the call is silently ignored. This allows instrumentation code to be added unconditionally without error handling.

Use \[Write] when you have direct access to an \[http.ResponseWriter] instead of a context with a session.

### func Write

```go
func Write(w http.ResponseWriter, entry Entry)
```

Write adds a timing entry directly to the response headers. If the entry fails validation, the call is silently ignored.

Use \[Record] when working with a context that contains a \[zen.Session], which is the common case in request handlers.


## Types

### type Entry

```go
type Entry struct {
	// Name identifies the operation being measured (e.g., "cache_get", "db_query").
	Name string
	// Attributes provides optional context (e.g., {"status": "miss", "table": "keys"}).
	Attributes map[string]string
	// Duration is the measured time span.
	Duration time.Duration
}
```

Entry represents a single timing measurement that can be serialized to the \[HeaderName] header.

Name follows Prometheus metric naming conventions: it must start with a letter or underscore, and may contain letters, digits, underscores, and colons.

Attributes are optional key-value pairs providing context about the measurement. Keys follow the same naming rules as Name. Values must be non-empty, contain only printable ASCII (0x21-0x7E), and exclude the characters '{', '}', ',', and '='.

Duration must be non-negative. When serialized, it uses the smallest unit that avoids sub-unit fractions (e.g., 1500µs becomes "1.5ms", not "1500us").

#### func ParseEntries

```go
func ParseEntries(input string) ([]Entry, error)
```

ParseEntries parses a comma-separated string of timing entries. This is useful for parsing concatenated header values when multiple entries are joined in a single string. Commas inside attribute braces are treated as attribute separators, not entry separators.

Returns an error if the input is empty, contains whitespace, has malformed entries, or contains empty entries between commas.

#### func ParseEntry

```go
func ParseEntry(input string) (Entry, error)
```

ParseEntry parses a single timing entry from its string representation. The input must match the format: name\[{key=value,...}]=value+unit

Returns the zero Entry and an error if parsing fails. Possible errors include empty input, whitespace in input, invalid name or attribute format, missing or malformed duration value, and unsupported unit suffix.

Use \[ParseEntries] when parsing multiple comma-separated entries.

#### func (Entry) String

```go
func (e Entry) String() string
```

String formats the entry as an \[HeaderName] header value. The output follows the format documented in the package overview: name, optional attributes in braces, equals sign, and duration with unit suffix. Attributes are sorted alphabetically by key for deterministic output.

#### func (Entry) Validate

```go
func (e Entry) Validate() error
```

Validate checks that the entry conforms to the timing format specification. Returns nil if valid, or an error describing the first constraint violation.

Validation rules:

  - Name must be non-empty and follow Prometheus naming (start with letter/underscore)
  - Duration must be non-negative
  - Attribute keys must follow name rules; values must be non-empty printable ASCII excluding '{', '}', ',', '='

