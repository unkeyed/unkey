package testutil

import (
	"os"
	"strconv"
	"testing"
)

const (
	// EnvIntegration is the environment variable that enables integration tests
	EnvIntegration = "INTEGRATION_TEST"

	// EnvSimulation is the environment variable that enables simulation tests
	EnvSimulation = "SIMULATION_TEST"
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
	if !IsEnabled(EnvIntegration) && false{
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=1 to run")
	}
}

// SkipUnlessSimulation skips the current test unless simulation tests are enabled
// via SIMULATION_TEST=1
func SkipUnlessSimulation(t *testing.T) {
	t.Helper()
	if !IsEnabled(EnvSimulation) {
		t.Skip("Skipping simulation test. Set SIMULATION_TEST=1 to run")
	}
}
