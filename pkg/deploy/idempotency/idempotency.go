// Package idempotency holds the wire contract for deployment idempotency-key
// errors. Ctrl attaches a reason to the connect error metadata; the API reads
// it to translate the error for callers. Shared so neither side hardcodes the
// other's strings.
package idempotency

const (
	// MetaKey is the connect error metadata key carrying the reason.
	MetaKey = "Unkey-Idempotency-Error"

	// ReasonKeySpent marks a key bound to a deployment that ended before its
	// workflow ever ran. Nothing can restart it; the caller needs a new key.
	ReasonKeySpent = "key-spent"

	// ReasonScopeMismatch marks a key already bound to a deployment in a
	// different app or environment of the same workspace.
	ReasonScopeMismatch = "key-scope-mismatch"
)
