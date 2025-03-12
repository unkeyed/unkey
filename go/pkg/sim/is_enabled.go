package sim

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// enableEnvKey is the environment variable name that controls whether simulation tests run.
// When not set or set to "false", simulation tests will be skipped.
const enableEnvKey = "SIMULATON_TEST"

// CheckEnabled verifies if simulation tests should run based on environment configuration.
// It skips the current test if simulations are not enabled.
//
// Simulation tests can be resource-intensive and time-consuming, so this function
// provides a convenient way to conditionally run them, particularly in CI environments
// where you might want to run simulations only in specific scenarios.
//
// To enable simulation tests, set the environment variable:
//
//	SIMULATON_TEST=true
//
// Example usage:
//
//	func TestComplexSimulation(t *testing.T) {
//	    sim.CheckEnabled(t) // Skip if simulations not enabled
//	    // ... rest of the test
//	}
func CheckEnabled(t *testing.T) {
	t.Helper()

	// Check if the environment variable is set
	env := os.Getenv(enableEnvKey)
	if env == "" {
		t.Skipf("%s not set", enableEnvKey)
	}

	// Parse the boolean value
	enabled, err := strconv.ParseBool(env)
	require.NoError(t, err)

	// Skip the test if simulations are not enabled
	if !enabled {
		t.Skipf("Simulation test is not enabled, set %s=true to run simulations", enableEnvKey)
	}
}
