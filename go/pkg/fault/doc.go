// Package fault provides a clean, concise error handling system designed for
// building robust applications with rich error context and secure user messaging.
//
// The primary API consists of four essential functions:
//   - fault.Wrap: Wraps errors while capturing source locations
//   - fault.Internal: Adds debugging information (not exposed to users)
//   - fault.Public: Adds user-safe messages (safe for API responses)
//   - fault.Code: Adds error classification codes
//
// Key features:
//   - Clean separation between internal debugging and user-facing messages
//   - Concise, ergonomic API with short function names
//   - Automatic source location tracking throughout error chains
//   - Flexible error classification system
//   - Safe error chain unwrapping and inspection
//
// Basic usage:
//
//	err := fault.New("database connection failed")
//	if err != nil {
//	    return fault.Wrap(err,
//	        fault.Code(DATABASE_ERROR),
//	        fault.Internal("connection timeout after 30s"),
//	        fault.Public("Service temporarily unavailable"),
//	    )
//	}
//
// Individual message control:
//
//	// Only internal debugging information
//	fault.Wrap(err, fault.Internal("detailed debug context"))
//
//	// Only user-facing message
//	fault.Wrap(err, fault.Public("Please try again later"))
//
//	// Combine as needed
//	fault.Wrap(err,
//	    fault.Internal("connection failed to 192.168.1.1:5432"),
//	    fault.Public("Database unavailable"),
//	    fault.Code(DATABASE_ERROR),
//	)
//
// Legacy API: The following functions are deprecated but still available:
// WithDesc, WithInternalDesc, WithPublicDesc, WithCode
package fault
