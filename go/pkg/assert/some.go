package assert

// Some checks multiple assertions and returns nil if at least one assertion passes.
// If all assertions fail, it returns the first error encountered.
//
// Unlike All, which ensures all assertions pass, Some only requires that at least
// one assertion succeeds. This is useful for validating conditions where multiple
// alternatives are acceptable.
//
// Example:
//
//	// Check if user has any required role
//	err := assert.Some(
//	    assert.Equal(user.Role, "admin"),
//	    assert.Equal(user.Role, "editor"),
//	    assert.Equal(user.Role, "manager"),
//	)
//	if err != nil {
//	    return fault.Wrap(err, fault.Internal("insufficient permissions"), fault.Public("User lacks required role"))
//	}
func Some(errs ...error) error {
	var firstErr error
	for _, err := range errs {
		if err == nil {
			// At least one assertion passed
			return nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
