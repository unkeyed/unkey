// Package clock provides a flexible interface for time-related operations,
// allowing for consistent time handling in both production and test environments.
//
// The key benefit of this package is that it enables deterministic testing of
// time-dependent code by providing mock implementations that can be controlled
// in tests, while using the system clock in production.
//
// The package offers:
// - A standard interface for getting the current time
// - A production implementation that uses the system clock
// - A test implementation that allows precise control over time
//
// Example:
//
//	// In production code:
//	func ProcessWithExpiry(clock clock.Clock, data []Item) []Item {
//	    now := clock.Now()
//	    result := make([]Item, 0)
//	    for _, item := range data {
//	        if item.ExpiresAt.After(now) {
//	            result = append(result, item)
//	        }
//	    }
//	    return result
//	}
//
//	// In production:
//	processor := &Processor{clock: clock.New()}
//
//	// In tests:
//	func TestProcessWithExpiry(t *testing.T) {
//	    clock := clock.NewTestClock()
//
//	    // Set a fixed time for deterministic testing
//	    fixedTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
//	    clock.Set(fixedTime)
//
//	    // Create test data that expires at different times
//	    data := []Item{
//	        {ID: "1", ExpiresAt: fixedTime.Add(-time.Hour)},  // expired
//	        {ID: "2", ExpiresAt: fixedTime.Add(time.Hour)},   // not expired
//	    }
//
//	    result := ProcessWithExpiry(clock, data)
//
//	    // Should only contain the non-expired item
//	    assert.Len(t, result, 1)
//	    assert.Equal(t, "2", result[0].ID)
//
//	    // Advance time past the expiration of the second item
//	    clock.Tick(2 * time.Hour)
//
//	    // Now all items should be expired
//	    result = ProcessWithExpiry(clock, data)
//	    assert.Empty(t, result)
//	}
package clock
