package validation

import "regexp"

// envVarNameRegex matches a valid environment variable name: it must start with
// a letter or underscore and then contain only letters, digits, and
// underscores. This is intentionally stricter than the K8s Secret data-key
// charset (which also permits dots and hyphens), because the key is consumed as
// an environment variable in two places that both reject other characters:
//
//   - At build time the variables are written to a .env file and loaded with
//     `set -a && . /run/secrets/.env && set +a`. POSIX `.` only assigns names
//     that are valid shell identifiers.
//   - At runtime the Secret is projected into the container with envFrom, which
//     silently drops any key that is not a valid env var name.
//
// Allowing dots or hyphens lets users save keys that never actually reach the
// container, so we reject them at the input boundary instead.
var envVarNameRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrMsgInvalidEnvVarKey is the human-readable explanation used in validation errors.
const ErrMsgInvalidEnvVarKey = "must start with a letter or underscore and contain only letters, numbers, and underscores"

// IsValidEnvVarKey reports whether key is usable as an environment variable
// name in both the build (.env sourcing) and runtime (envFrom) paths.
func IsValidEnvVarKey(key string) bool {
	return envVarNameRegex.MatchString(key)
}
