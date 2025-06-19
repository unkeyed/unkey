package assert

// All combines multiple assertion checks into a single operation, returning
// the first encountered error (if any). This function allows for concise validation
// of multiple conditions without repetitive error checking.
//
// All stops checking at the first error it encounters.
//
// Example:
//
//	err := assert.All(
//	    assert.NotNil(user),
//	    assert.NotEmpty(user.ID),
//	    assert.True(user.IsActive, "user must be active"),
//	)
//	if err != nil {
//	    return fault.Wrap(err, fault.Internal("invalid user"), fault.Public("User validation failed"))
//	}
func All(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
