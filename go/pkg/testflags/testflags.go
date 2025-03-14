package testflags

import (
	"os"
	"strconv"
	"testing"
)

const (
	// EnvIntegration is the environment variable that enables integration tests
	EnvIntegration = "INTEGRATION_TEST"
)

// IsEnabled checks if a specific test flag is enabled via environment variable
func IsEnabled(envVar string) bool {
	val, exists := os.LookupEnv(envVar)
	if !exists {
		return false
	}

	enabled, err := strconv.ParseBool(val)
	if err != nil {
		// If the value isn't a valid boolean, treat non-empty strings as "true"
		return val != ""
	}

	return enabled
}

// SkipUnlessIntegration skips the current test unless integration tests are enabled
// via INTEGRATION_TEST=1
func SkipUnlessIntegration(t *testing.T) {
	t.Helper()
	if !IsEnabled(EnvIntegration) {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}
}
