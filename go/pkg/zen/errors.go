package zen

import "github.com/unkeyed/unkey/go/pkg/fault"

// Error tags for common error scenarios.
var (
	NotFoundError = fault.Tag("NOT_FOUND_ERROR")

	// DatabaseError represents errors that occur during database operations.
	// This error tag is used when database operations fail, such as connection
	// issues, query failures, or data integrity problems.
	DatabaseError = fault.Tag("DATABASE_ERROR")
)
