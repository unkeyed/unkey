package streamschema

import (
	"fmt"
)

// Reason classifies why [Schema.Validate] rejected its input.
type Reason int

const (
	// ReasonSchema means a schema constraint failed. The rest of the input
	// may not have been parsed.
	ReasonSchema Reason = iota + 1
	// ReasonSyntax means the input is not strict JSON. See the package
	// documentation for the rules beyond RFC 8259.
	ReasonSyntax
	// ReasonLimit means the input exceeded one of the [Limits].
	ReasonLimit
)

// String returns the lowercase name of r.
func (r Reason) String() string {
	switch r {
	case ReasonSchema:
		return "schema"
	case ReasonSyntax:
		return "syntax"
	case ReasonLimit:
		return "limit"
	default:
		return fmt.Sprintf("Reason(%d)", int(r))
	}
}

// ValidationError describes the first failure found in a document. Message
// quotes schema values only, never input content, so it is safe to return to
// the client that sent the input.
type ValidationError struct {
	// Reason classifies the failure.
	Reason Reason
	// InstanceLocation is an RFC 6901 JSON Pointer to the value in the input
	// that failed; the empty string is the root. For syntax and limit failures
	// it is the innermost value or member being read when parsing stopped.
	InstanceLocation string
	// KeywordLocation is an RFC 6901 JSON Pointer into the schema document to
	// the keyword that failed, such as "/properties/name/maxLength". It is empty
	// for syntax and limit failures.
	KeywordLocation string
	// Offset is a byte offset in the input. Schema failures point to the
	// failing token's start; syntax failures can point inside a token.
	Offset int64
	// Message is a short human-readable description.
	Message string
}

// Error formats e for logs.
func (e *ValidationError) Error() string {
	if e.KeywordLocation != "" {
		return fmt.Sprintf("%s error at offset %d, instance %q, keyword %q: %s", e.Reason, e.Offset, e.InstanceLocation, e.KeywordLocation, e.Message)
	}
	return fmt.Sprintf("%s error at offset %d, instance %q: %s", e.Reason, e.Offset, e.InstanceLocation, e.Message)
}

// CompileErrorKind classifies why [Compile] rejected a schema document.
type CompileErrorKind int

const (
	// CompileUnsupported means the document uses a keyword or form this
	// package does not implement. The rest of the schema may not be checked.
	CompileUnsupported CompileErrorKind = iota + 1
	// CompileInvalid means the document is not a valid schema, for example
	// malformed JSON, a duplicate key, or a keyword with a value of the wrong
	// type.
	CompileInvalid
	// CompileLimit means the document exceeded one of the schema limits in
	// [Limits].
	CompileLimit
)

// String returns the lowercase name of k.
func (k CompileErrorKind) String() string {
	switch k {
	case CompileUnsupported:
		return "unsupported"
	case CompileInvalid:
		return "invalid"
	case CompileLimit:
		return "limit"
	default:
		return fmt.Sprintf("CompileErrorKind(%d)", int(k))
	}
}

// CompileError describes the first problem [Compile] found in a schema
// document. Keywords are checked depth-first in sorted key order, so the
// reported problem is deterministic.
type CompileError struct {
	// Kind classifies the problem.
	Kind CompileErrorKind
	// KeywordLocation is an RFC 6901 JSON Pointer into the schema document.
	KeywordLocation string
	// Message is a short human-readable description.
	Message string
	// Err is the [*ValidationError] from parsing the document when the
	// problem is in its JSON syntax or size, and nil otherwise.
	Err error
}

// Error formats e for logs.
func (e *CompileError) Error() string {
	return fmt.Sprintf("%s schema at %q: %s", e.Kind, e.KeywordLocation, e.Message)
}

// Unwrap returns the underlying parse error, if any.
func (e *CompileError) Unwrap() error {
	return e.Err
}
