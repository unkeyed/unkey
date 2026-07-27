// Package validation validates HTTP requests against an OpenAPI specification
// and turns the failures into an error response a developer can act on.
//
// # What a caller gets
//
// A rejected request comes back with a short summary and one entry per problem.
// Each entry has a location that points into the request the way a developer
// thinks about it, and a message that reads as a predicate about that location:
//
//	body.key                  is required
//	body.variables[1].key     must match the pattern '^[A-Za-z_][A-Za-z0-9_]*$'
//	body.variables[0].value   must be at most 16384 characters long
//	query.limit               must be at most 100
//
// Entries are sorted by location and then message, because the underlying
// library walks a map and would otherwise report the same request's problems in a
// different order each time.
//
// # Why none of it echoes the request
//
// These strings are returned to the caller and written to the ClickHouse request
// log. A caller who puts a secret in a field that has a pattern, or sends it
// under a misspelled property name, must not end up with a stored copy of it, and
// the same package validates customer specifications inside frontline, where a
// credential in a header, a query parameter, or a URL path is entirely normal.
//
// So the rule is that every string in the response is either a literal from this
// package or a token that came from the specification:
//
//   - Messages are built from the typed error kinds of the JSON Schema library
//     (kind.Pattern, kind.MaxLength, kind.Enum, and the rest). Each kind carries
//     both the constraint and the offending value; describeKind reads only the
//     constraint. An unrecognised kind degrades to a fixed sentence rather than
//     to the library's prose.
//   - Locations are instance paths, but a segment naming a property is only
//     emitted when the same failure's schema pointer declares that property. A
//     caller-chosen name, such as a key inside a free-form object, becomes '*'.
//   - The summary is assembled from the request method, the matched spec path
//     template, the parameter's declared name, and the failure category. The
//     library's own summary interpolates the raw URL path, which in a customer
//     specification can be the credential itself.
//   - Every token this package treats as specification data is length-bounded and
//     lists of them are count-bounded, so a future library version that starts
//     putting a payload in a Want field loses detail instead of leaking.
//   - Nothing reads a Got field, a length, a magnitude, a property name from the
//     request, a Content-Type header, or a JSON decoder message.
//
// The three places that still match the library's prose are marked as such: they
// classify a failure or lift out a token the library read from the
// specification, and each falls back to a fixed sentence when the shape is not
// the one that was reviewed.
package validation
