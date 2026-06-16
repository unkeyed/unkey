package validation

import "regexp"

// envVarKeyRegex matches POSIX shell variable names. Environment variables
// exist to become process env: builds expose them to install/build commands,
// and the runtime injects them into the container. Names with dots or hyphens
// are unreachable from shells and most languages' env APIs, so they are
// rejected at every boundary rather than failing in confusing ways later.
//
// This is deliberately stricter than what Kubernetes accepts for Secret data
// keys; the old, laxer rule let users create variables their builds and apps
// could never read.
var envVarKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrMsgInvalidEnvVarKey is the human-readable explanation used in validation errors.
const ErrMsgInvalidEnvVarKey = "only letters, digits, and underscores are allowed, and the name must not start with a digit"

// IsValidEnvVarKey reports whether key is a valid environment variable name.
func IsValidEnvVarKey(key string) bool {
	return envVarKeyRegex.MatchString(key)
}
