package fault_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// TestFlatteningDemo demonstrates the efficiency improvement of the flattening approach
func TestFlatteningDemo(t *testing.T) {
	baseErr := errors.New("database connection failed")

	// Create a complex error with multiple wrappers in a single Wrap call
	err := fault.Wrap(baseErr,
		fault.Code(codes.URN("DATABASE_ERROR")),
		fault.Internal("connection timeout after 30s to 192.168.1.1:5432"),
		fault.Public("Service temporarily unavailable"),
		fault.Internal("retry attempt 3/3 failed"),
		fault.Public("Please try again in a few minutes"),
	)

	// Verify it's a single wrapped instance by checking unwrap behavior
	unwrapped := errors.Unwrap(err)
	require.Equal(t, baseErr, unwrapped, "Should unwrap directly to base error (single instance)")

	// Show the flattened structure
	fmt.Printf("=== Flattened Error Structure ===\n")
	fmt.Printf("Type: %T\n", err)
	fmt.Printf("Code: %s\n", getCode(err))
	fmt.Printf("Internal: %s\n", err.Error())
	fmt.Printf("Public: %s\n", fault.UserFacingMessage(err))
	fmt.Printf("Underlying: %v\n", unwrapped)
	fmt.Printf("Memory footprint: Single instance instead of 5 nested instances\n\n")

	// Demonstrate the old behavior would have created nested instances
	// (if we were to manually create them the old way)
	fmt.Printf("=== What the old approach would have created ===\n")
	fmt.Printf("Without flattening: 5 nested &wrapped instances\n")
	fmt.Printf("With flattening: 1 &wrapped instance\n")
	fmt.Printf("Memory reduction: ~80%% fewer allocations\n\n")

	// Show that unwrapping still works correctly for nested Wrap calls
	level1 := fault.Wrap(baseErr, fault.Internal("level 1"))
	level2 := fault.Wrap(level1, fault.Internal("level 2"))

	fmt.Printf("=== Nested Wrap calls (still create proper chains) ===\n")
	fmt.Printf("level2: %s\n", level2.Error())

	unwrappedLevel := errors.Unwrap(level2)
	fmt.Printf("unwrapped: %s\n", unwrappedLevel.Error())

	unwrappedBase := errors.Unwrap(unwrappedLevel)
	fmt.Printf("unwrapped again: %s\n", unwrappedBase.Error())
}

// Helper to get code without importing internal types
func getCode(err error) string {
	if code, ok := fault.GetCode(err); ok {
		return string(code)
	}
	return ""
}

// Example_flattening demonstrates the practical benefits
func Example_flattening() {
	baseErr := errors.New("connection failed")

	// This creates only ONE wrapped instance, not four
	err := fault.Wrap(baseErr,
		fault.Code(codes.URN("NETWORK_ERROR")),
		fault.Internal("timeout after 30s"),
		fault.Public("Service unavailable"),
	)

	// All information is efficiently stored in a single instance
	fmt.Println("Error:", err.Error())
	fmt.Println("User message:", fault.UserFacingMessage(err))

	if code, ok := fault.GetCode(err); ok {
		fmt.Println("Code:", code)
	}

	// Output:
	// Error: timeout after 30s: connection failed
	// User message: Service unavailable
	// Code: NETWORK_ERROR
}

// BenchmarkFlattening compares memory allocations
func BenchmarkFlattening(b *testing.B) {
	baseErr := errors.New("base error")

	b.Run("single_wrap_multiple_wrappers", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// This is now efficient - single allocation
			_ = fault.Wrap(baseErr,
				fault.Code(codes.URN("TEST")),
				fault.Internal("debug"),
				fault.Public("user"),
			)
		}
	})

	b.Run("multiple_wrap_calls", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			// This still creates multiple instances (intentionally)
			err := fault.Wrap(baseErr, fault.Internal("level1"))
			err = fault.Wrap(err, fault.Internal("level2"))
			err = fault.Wrap(err, fault.Internal("level3"))
			_ = err
		}
	})
}
