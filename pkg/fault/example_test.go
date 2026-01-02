package fault_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ExampleInternal demonstrates using Internal for debug-only context
func ExampleInternal() {
	// Simulate a database connection error
	dbErr := errors.New("connection refused")

	// Add internal debugging context without exposing details to users
	err := fault.Wrap(dbErr,
		fault.Code(codes.URN("DATABASE_ERROR")),
		fault.Internal("failed to connect to primary database at 192.168.1.100:5432"),
	)

	// Internal error message (for logging)
	fmt.Println("Internal error:", err.Error())

	// User-facing message (empty since no public description was set)
	fmt.Println("User message:", fault.UserFacingMessage(err))

	// Output:
	// Internal error: failed to connect to primary database at 192.168.1.100:5432: connection refused
	// User message:
}

// ExamplePublic demonstrates using Public for user-facing messages only
func ExamplePublic() {
	// Simulate a validation error
	validationErr := errors.New("field 'email' is required")

	// Add user-friendly message without internal debugging details
	err := fault.Wrap(validationErr,
		fault.Code(codes.URN("VALIDATION_ERROR")),
		fault.Public("Please provide a valid email address"),
	)

	// Internal error message (includes original error)
	fmt.Println("Internal error:", err.Error())

	// User-facing message (safe for API responses)
	fmt.Println("User message:", fault.UserFacingMessage(err))

	// Output:
	// Internal error: field 'email' is required
	// User message: Please provide a valid email address
}

// Example_mixedDescriptions demonstrates combining different description types
func Example_mixedDescriptions() {
	// Simulate a complex operation with multiple error layers
	baseErr := errors.New("network timeout")

	err := fault.Wrap(baseErr,
		fault.Code(codes.URN("NETWORK_ERROR")),
		fault.Internal("upstream service call failed"),
		fault.Public("Service temporarily unavailable"),
		fault.Internal("retry attempt 3/3 failed after 30s"),
		fault.Public("Please try again in a few minutes"),
	)

	// Internal error message (all internal context)
	fmt.Println("Internal error:", err.Error())

	// User-facing message (combined public messages)
	fmt.Println("User message:", fault.UserFacingMessage(err))

	// Output:
	// Internal error: retry attempt 3/3 failed after 30s: upstream service call failed: network timeout
	// User message: Please try again in a few minutes Service temporarily unavailable
}

// Example_conditionalDescriptions demonstrates conditional error enhancement
func Example_conditionalDescriptions() {
	processRequest := func(userID string, debug bool) error {
		// Simulate some processing error
		err := errors.New("processing failed")

		if debug {
			// Add detailed debugging information for development
			return fault.Wrap(err,
				fault.Internal(fmt.Sprintf("failed to process request for user %s at step 3", userID)),
				fault.Public("An error occurred while processing your request"),
			)
		} else {
			// Production: only user-facing message
			return fault.Wrap(err,
				fault.Public("An error occurred while processing your request"),
			)
		}
	}

	// Development mode
	devErr := processRequest("user123", true)
	fmt.Println("Dev error:", devErr.Error())
	fmt.Println("Dev user message:", fault.UserFacingMessage(devErr))

	// Production mode
	prodErr := processRequest("user123", false)
	fmt.Println("Prod error:", prodErr.Error())
	fmt.Println("Prod user message:", fault.UserFacingMessage(prodErr))

	// Output:
	// Dev error: failed to process request for user user123 at step 3: processing failed
	// Dev user message: An error occurred while processing your request
	// Prod error: processing failed
	// Prod user message: An error occurred while processing your request
}

// Example_errorHandling demonstrates a realistic error handling scenario
func Example_errorHandling() {
	// Simulate a service function that might fail in different ways
	authenticateUser := func(token string) error {
		if token == "" {
			return fault.New("missing authentication token",
				fault.Code(codes.URN("AUTH_MISSING")),
				fault.Public("Authentication required"),
			)
		}

		if token == "invalid" {
			return fault.New("token validation failed",
				fault.Code(codes.URN("AUTH_INVALID")),
				fault.Internal(fmt.Sprintf("invalid token format: %q", token)),
				fault.Public("Invalid authentication credentials"),
			)
		}

		if token == "expired" {
			return fault.New("token expired",
				fault.Code(codes.URN("AUTH_EXPIRED")),
				fault.Public("Your session has expired. Please log in again"),
			)
		}

		return nil
	}

	// Test different scenarios
	testCases := []string{"", "invalid", "expired", "valid"}

	for _, token := range testCases {
		if err := authenticateUser(token); err != nil {
			// Log internal details
			log.Printf("Auth error: %v", err)

			// Send user-friendly message to client
			userMsg := fault.UserFacingMessage(err)
			if userMsg != "" {
				fmt.Printf("Client response for '%s': %s\n", token, userMsg)
			}
		} else {
			fmt.Printf("Client response for '%s': Authentication successful\n", token)
		}
	}

	// Output:
	// Client response for '': Authentication required
	// Client response for 'invalid': Invalid authentication credentials
	// Client response for 'expired': Your session has expired. Please log in again
	// Client response for 'valid': Authentication successful
}
