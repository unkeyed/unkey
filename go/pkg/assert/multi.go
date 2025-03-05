package assert

// Multi combines multiple assertion checks into a single operation, returning
// the first encountered error (if any). This function allows for concise validation
// of multiple conditions without repetitive error checking.
//
// Multi stops checking at the first error it encounters.
//
// Example:
//
//	err := assert.Multi(
//	    assert.NotNil(user),
//	    assert.NotEmpty(user.ID),
//	    assert.True(user.IsActive, "user must be active"),
//	)
//	if err != nil {
//	    return fault.Wrap(err, fault.WithDesc("invalid user", "User validation failed"))
//	}
func Multi(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
