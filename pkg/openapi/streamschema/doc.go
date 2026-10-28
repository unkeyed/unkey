// Package streamschema validates one JSON document read from an [io.Reader]
// against a compiled subset of JSON Schema 2020-12, the dialect of the OpenAPI
// 3.1 Schema Object, without holding the document in memory.
//
// The validator is a recursive-descent parser driven by the schema: the schema
// node for the current value selects which checks run while the bytes stream
// past. A string is never materialized; its code points are counted as they are
// decoded, so a multi-gigabyte string costs the same heap as an empty one. The
// parser is preferred over a token or event API because a token API either
// buffers whole strings (as encoding/json and encoding/json/jsontext do) or
// splits them into chunk events that every consumer has to reassemble.
//
// # Supported subset
//
// [Compile] accepts boolean schemas and these keywords: type, properties,
// required, additionalProperties, minLength, maxLength, items (single schema),
// minItems, maxItems, minimum, maximum, exclusiveMinimum, and exclusiveMaximum
// (numeric form). Type-specific keywords constrain only instances of their type,
// as JSON Schema requires, so maxLength has no effect on a number even when type
// is absent. An explicit list of annotations (title, description, $comment,
// default, examples, example, deprecated, readOnly, writeOnly, format,
// externalDocs) is accepted and has no effect; format is an annotation in 2020-12
// and the libopenapi-validator configuration this package may replace does not
// assert it. $schema is accepted only at the root and only for the 2020-12 or
// OpenAPI 3.1 dialects. Every other keyword, including $ref, $defs, enum,
// pattern, and the combinators, fails compilation with [CompileUnsupported], so
// an assertion is never silently dropped.
//
// # Strictness
//
// Input must be RFC 8259 JSON and also meet the I-JSON rules (RFC 7493) that
// matter for a validator placed in front of another parser: valid UTF-8 with no
// byte order mark, no escaped unpaired surrogates, and no duplicate member names
// after escapes are decoded. Lenient decoders resolve those cases differently
// (encoding/json keeps the last duplicate and replaces invalid UTF-8), so
// accepting them would let a validated body mean something else downstream.
// Schemas are held to the same rules.
//
// Numbers are compared exactly as normalized decimals (coefficient digits plus a
// base-10 exponent) and are never converted to float64 or expanded into powers
// of ten. A number whose literal exponent exceeds one billion in magnitude is
// rejected with [ReasonLimit] rather than approximated.
//
// # Resource bounds
//
// A compiled [Schema] is immutable and safe for concurrent use. Its heap is
// bounded by the schema limits in [Limits]: at most SchemaNodesMax nodes, each
// holding keyword values copied from a document of at most SchemaBytesMax bytes.
//
// One call to [Schema.Validate] holds Go heap bounded by its limits, not by the
// input: a fixed 32 KiB read buffer; member-name and number buffers whose length
// never exceeds KeyBytesMax and NumberBytesMax (capacity may reach about twice
// that through append growth); the member names of open objects, whose
// accounted size is capped by KeysRetainedBytesMax; up to DepthMax cleared
// name sets of at most 64 entries kept for reuse; and one path segment and one
// recursion frame per open container up to DepthMax. Each retained name is
// charged its decoded length plus 64 bytes, so many empty or short names cannot
// escape the budget; the charge approximates, and does not measure, Go map
// overhead. Garbage from closed objects is not held and is left to the GC. The
// bound covers heap held by the validator, not total process RAM, memory the
// reader uses, or goroutine stack, which grows with DepthMax.
//
// Validate stops at the first failure. Context cancellation is checked each time
// the read buffer is refilled, which happens at least once per 32 KiB consumed.
package streamschema
