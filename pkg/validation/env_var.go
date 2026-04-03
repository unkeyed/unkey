package validation

import "regexp"

// envVarKeyRegex matches valid K8s Secret data keys.
// Mirrors the regex K8s uses internally: alphanumeric characters, '-', '_', or '.'.
var envVarKeyRegex = regexp.MustCompile(`^[-._a-zA-Z0-9]+$`)

// IsValidEnvVarKey reports whether key is a valid K8s Secret data key.
func IsValidEnvVarKey(key string) bool {
	return envVarKeyRegex.MatchString(key)
}
