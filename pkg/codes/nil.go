package codes

// Nil represents a nil or unknown error code. It's used as a default value
// when a specific error code is not available or applicable. This helps
// maintain consistency in error handling even for unclassified errors.
var Nil = Code{
	System:   SystemNil,
	Category: "unknown",
	Specific: "unknown",
}
