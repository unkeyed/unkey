// Package fault provides a comprehensive error handling system designed for
// building robust applications with rich error context and debugging
// capabilities. It implements a multi-layered error handling approach with
// several key features:
//
//   - Dual-message error pattern: Separate internal (debug) and
//     public (user-facing) error messages.
//   - Error chain tracking: Capture and preserve complete error context with
//     source locations.
//   - Error tagging: Flexible error classification system for consistent error
//     handling.
//   - Stack trace preservation: Automatic capture of error locations throughout
//     the chain.
//
// It builds upon Go's standard error handling patterns while adding structured
// context and safety mechanisms.
//
// Basic usage:
//
//	err := fault.New("database connection failed")
//	if err != nil {
//	    return fault.Wrap(err,
//	        fault.WithTag(DATABASE_ERROR),
//	        fault.WithDesc("init failed", "Service temporarily unavailable"),
//	    )
//	}
package fault
