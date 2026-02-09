// Package timing defines a small, strict header format for server-side timing
// data and provides helpers to record it through zen sessions.
//
// The package exists because Server-Timing is standardized but too limited for
// Unkey's debugging needs. We want a header that remains readable in raw HTTP
// responses, supports optional attributes, and is simple to parse without the
// complexity of quoted values or browser integration.
//
// # Format
//
// Each entry is encoded as a single header value for the X-Unkey-Timing header.
// A typical example is:
//
//	X-Unkey-Timing: cache_get{status=miss,cache=ApiByID}=1.25ms
//
// An entry has a name, optional attributes in braces, and a numeric value with a
// unit suffix. One header value should encode one entry to keep parsing small and
// avoid quoting edge cases. Whitespace is not permitted anywhere in the entry.
// Names and label keys follow the same grammar as Prometheus metric names. Label
// values are unquoted, must be non-empty, and must not contain whitespace or any
// of the characters '{', '}', ',', or '='. Values are unsigned numbers and cannot
// use sign or exponent notation.
//
// # Value and Units
//
// Values are integers or floating-point numbers followed by a unit suffix that
// maps to a [time.Duration] unit. Supported units are ns, us, ms, and s. The
// decimal must have digits on both sides if present.
//
// # Examples
//
// Valid entries:
//
//	X-Unkey-Timing: cache_get=1ms
//	X-Unkey-Timing: cache_get{status=miss}=1.25ms
//	X-Unkey-Timing: db_query{table=keys,status=miss}=53ms
//
// Invalid entries:
//
//	X-Unkey-Timing: cache_get{status=}=1ms
//	X-Unkey-Timing: cache_get=.5ms
//	X-Unkey-Timing: cache_get=1e3ms
//	X-Unkey-Timing: cache_get{status=miss,reason=too slow}=1ms
//
// # Usage
//
// Callers record entries directly on the current request context. If a zen
// session is present, the entry is serialized and added to the response headers.
// This keeps observability wiring close to the request without exposing sessions
// to application code.
//
//	 timing.Record(ctx, timing.Entry{
//		Name:     "cache_get",
//		Duration: 1500 * time.Microsecond,
//		Attributes: map[string]string{
//			"status": "miss",
//		},
//	})
//
// If you need to read these values later, use [ParseEntry] or [ParseEntries] on
// the header values.
//
// # ABNF
//
// The grammar is expressed in ABNF as follows:
//
// entry       = name [labels] "=" value unit
// labels      = "{" label *("," label) "}"
// label       = name "=" label-value
// name        = name-start *(name-char)
// name-start  = ALPHA / "_"
// name-char   = ALPHA / DIGIT / "_" / ":"
// value       = 1*DIGIT ["." 1*DIGIT]
// unit        = "ns" / "us" / "ms" / "s"
// label-value = 1*(label-char)
// label-char  = %x21-7E
//
// The label-char set excludes '{', '}', ',', and '=' per the rules above.
//
// # Design Goals
//
// The format is readable in raw HTTP headers, cheap to parse in hot paths, and
// extensible through optional attributes.
//
// # Non-goals
//
// The format does not aim for browser parsing or Prometheus scrape
// compatibility.
package timing
