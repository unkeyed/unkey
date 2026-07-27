// Package redaction strips sensitive values out of JSON bodies before they are
// persisted to request logs.
//
// Redaction is position-driven. A [Redactor] holds a set of paths and replaces a
// member's value only where the path leading to it matches. Paths come from the
// OpenAPI specification via [PathsFromSpec], so annotating a property with
// `x-unkey-redact: true` is the only step needed to keep its value out of
// storage, and two properties that happen to share a name stay independent:
//
//	variables[].value        the environment variable value in a request body
//	data.variables[].value   the same value on the way back out
//
// Because paths are anchored at the root of the body,
// `{"value":"x","xyz":{"value":"y"}}` redacts only the member whose path was
// annotated.
package redaction
