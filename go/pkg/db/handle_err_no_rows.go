package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/fault"
)

// HandleErr processes database errors and wraps them with appropriate domain context.
// It categorizes database errors into specific types using fault tags.
//
// Error handling:
//   - Returns nil if err is nil
//   - For [sql.ErrNoRows], returns a [fault.Error] with [fault.NOT_FOUND] tag
//   - For all other database errors, returns a [fault.Error] with [fault.DATABASE_ERROR] tag
//
// Parameters:
//   - err: The error to process, typically returned from a database operation
//   - resource: A string describing the resource type being queried (e.g. "user", "key")
//
// Returns:
//   - nil if err is nil
//   - A [fault.Error] with appropriate tag based on the error type
//
// Thread safety:
//
//	This function is stateless and safe for concurrent use.
//
// Examples:
//
//	// Basic usage with a database query
//	user, err := db.QueryRow("SELECT * FROM users WHERE id = ?", userID).Scan(&userData)
//	if err != nil {
//	    return db.HandleErr(err, "user")
//	}
//
//	// Error handling pattern with specific error type checking
//	func GetProduct(id string) (*Product, error) {
//	    var p Product
//	    err := db.QueryRow("SELECT * FROM products WHERE id = ?", id).Scan(&p.ID, &p.Name)
//	    if err != nil {
//	        return nil, db.HandleErr(err, "product")
//	    }
//	    return &p, nil
//	}
//
//	// Checking for specific error types from caller code
//	product, err := GetProduct(productID)
//	if err != nil {
//	    if fault.HasTag(err, fault.NOT_FOUND) {
//	        // Handle not found case
//	        return nil, fmt.Errorf("product %s not found", productID)
//	    }
//	    if fault.HasTag(err, fault.DATABASE_ERROR) {
//	        // Handle database error case
//	        log.Error().Err(err).Msg("database error occurred")
//	        return nil, fmt.Errorf("internal server error")
//	    }
//	    // Handle other errors
//	    return nil, err
//	}
//
// See also:
//   - [fault.Wrap] for more information on how errors are wrapped
//   - [fault.WithTag] for adding semantic tags to errors
//   - [fault.WithDesc] for adding user-friendly error descriptions
//   - [errors.Is] for checking error types
func HandleErr(err error, resource string) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fault.Wrap(err,
			fault.WithTag(fault.NOT_FOUND),
			fault.WithDesc(fmt.Sprintf("%s not found", resource), fmt.Sprintf("%s does not exist.", resource)),
		)
	default:
		return fault.Wrap(err,
			fault.WithTag(fault.DATABASE_ERROR),
			fault.WithDesc(fmt.Sprintf("Database error for %s", resource), "An error occurred while accessing the database."),
		)
	}
}
