package assert

// NotNilAndNotZero asserts that the provided value is both not nil and not its zero value.
// This is useful for validating pointer types and interface values that should be
// properly initialized.
//
// For most types, this is equivalent to just NotZero since nil pointers, nil interfaces,
// and nil slices/maps are all zero values. However, this function provides clearer
// semantics when you specifically want to check both conditions.
//
// Example:
//
//	// Validate that a database interface is both provided and initialized
//	if err := assert.NotNilAndNotZero(db, "Database must be provided and initialized"); err != nil {
//	    return fault.Wrap(err, fault.Internal("database validation failed"))
//	}
//
//	// Validate that a config pointer is both not nil and has values set
//	if err := assert.NotNilAndNotZero(config, "Config must be provided and initialized"); err != nil {
//	    return fault.Wrap(err, fault.Internal("config validation failed"))
//	}
func NotNilAndNotZero[T comparable](value T, message ...string) error {
	// For comparable types, nil pointers, nil interfaces, nil slices/maps
	// are all zero values, so NotZero effectively handles both checks.
	// We call NotNil first for interface{} values to potentially provide
	// a more specific "nil" error message, then NotZero for the comprehensive check.

	// Check for interface{} nil values first
	if err := NotNil(any(value), message...); err != nil {
		return err
	}

	// Then check if zero value (which includes typed nil pointers)
	if err := NotZero(value, message...); err != nil {
		return err
	}

	return nil
}
