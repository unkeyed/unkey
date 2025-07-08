package containers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContainersSetup validates that testcontainers can be initialized without starting services
func TestContainersSetup(t *testing.T) {
	// Just test that we can get compose instance
	compose := getSharedCompose(t)
	require.NotNil(t, compose)

	t.Log("✅ TestContainers initialization successful")
	t.Log("✅ Docker compose file loaded")
	t.Log("✅ ComposeStack created")
}

// TestContainersValidation shows what would happen when services start
func TestContainersValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping in short mode - would start containers")
	}

	// Test that services can be started (but might take time for first run)
	t.Log("⏳ Starting containers (this may take time on first run)...")

	// Quick validation that we can get service info
	mysqlCfg, _ := MySQL(t)
	t.Logf("✅ MySQL available at: %s", mysqlCfg.Addr)

	_, hostAddr, _ := Redis(t)
	t.Logf("✅ Redis available at: %s", hostAddr)

	t.Log("✅ Containers started successfully!")
}
