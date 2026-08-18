// Package validation provides reusable input validation functions for the
// Unkey platform.
//
// It includes validators for:
//   - Environment variable keys (Kubernetes Secret data key format)
//   - Portal configuration slugs (human-readable identifiers)
//
// All validators are pure functions with no external dependencies, making them
// suitable for use in handlers, tests, and property-based testing.
//
// Slug validation example:
//
//	if !validation.ValidateSlug(req.Slug) {
//	    return fault.New("invalid slug",
//	        fault.Code(codes.App.Validation.InvalidInput.URN()),
//	        fault.Public(validation.ErrMsgInvalidSlug),
//	    )
//	}
package validation
