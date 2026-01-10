package api_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault/keys"
	"github.com/unkeyed/unkey/svc/api"
)

// TestContextCancellation verifies that the API server responds properly to context cancellation
func TestContextCancellation(t *testing.T) {

	// Use testcontainers for dynamic service management
	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "unkey"
	dbDsn := mysqlCfg.FormatDSN()
	redisUrl := dockertest.Redis(t)

	// Create ephemeral listener
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "Failed to create ephemeral listener")

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	// Configure the API server
	config := api.Config{
		Platform:                "test",
		Image:                   "test",
		Listener:                ln,
		Region:                  "test-region",
		Clock:                   nil, // Will use real clock
		InstanceID:              uid.New(uid.InstancePrefix),
		RedisUrl:                redisUrl,
		ClickhouseURL:           "",
		DatabasePrimary:         dbDsn,
		DatabaseReadonlyReplica: "",
		OtelEnabled:             false,
		VaultMasterKeys:         []string{masterKey},
	}

	// Create a channel to receive the result of the Run function
	resultCh := make(chan error, 1)

	// Start the API server in a goroutine
	go func() {
		runErr := api.Run(ctx, config)

		if runErr != nil {
			// it's really hard to get this error cause the test fails before we read from the channel
			t.Logf("Error from run: %s", runErr.Error())
		}
		resultCh <- runErr
	}()

	// Wait for the server to start up
	require.Eventually(t, func() bool {
		res, livenessErr := http.Get(fmt.Sprintf("http://%s/v2/liveness", ln.Addr()))
		if livenessErr != nil {
			return false
		}
		defer res.Body.Close()
		return res.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond, "API server failed to start")

	// Verify the server is running
	t.Log("API server started successfully")

	// Cancel the context to trigger shutdown
	cancel()

	// Wait for the server to shut down and check the result
	select {
	case err = <-resultCh:
		// The Run function should exit without error when context is canceled
		require.NoError(t, err, "API server should shut down gracefully when context is canceled")
		t.Log("API server shut down successfully")
	case <-time.After(90 * time.Second):
		t.Fatal("API server failed to shut down within the expected time")
	}

	// Verify the server is no longer responding
	_, err = http.Get(fmt.Sprintf("http://%s/v2/liveness", ln.Addr()))
	require.Error(t, err, "Server should no longer be responding after shutdown")
}
