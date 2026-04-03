package validation

import "regexp"

// envVarKeyRegex mirrors the regex K8s uses for Secret data keys.
var envVarKeyRegex = regexp.MustCompile(`^[-._a-zA-Z0-9]+$`)

// ErrMsgInvalidEnvVarKey is the human-readable explanation used in validation errors.
const ErrMsgInvalidEnvVarKey = "only letters, numbers, hyphens, underscores, and dots are allowed"

// IsValidEnvVarKey reports whether key is a valid K8s Secret data key.
func IsValidEnvVarKey(key string) bool {
	return envVarKeyRegex.MatchString(key)
}
