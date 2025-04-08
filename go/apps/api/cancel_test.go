package api_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api"
	"github.com/unkeyed/unkey/go/pkg/port"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// TestContextCancellation verifies that the API server responds properly to context cancellation
func TestContextCancellation(t *testing.T) {

	// Create a containers instance for database
	containers := containers.New(t)
	dbDsn, _ := containers.RunMySQL()
	_, redisUrl, _ := containers.RunRedis()
	// Get free ports for the node
	portAllocator := port.New()
	httpPort := portAllocator.Get()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Configure the API server
	config := api.Config{
		Platform:                "test",
		Image:                   "test",
		HttpPort:                httpPort,
		Region:                  "test-region",
		Clock:                   nil, // Will use real clock
		InstanceID:              uid.New(uid.InstancePrefix),
		RedisUrl:                redisUrl,
		ClickhouseURL:           "",
		DatabasePrimary:         dbDsn,
		DatabaseReadonlyReplica: "",
		OtelEnabled:             false,
	}

	// Create a channel to receive the result of the Run function
	resultCh := make(chan error, 1)

	// Start the API server in a goroutine
	go func() {
		err := api.Run(ctx, config)

		if err != nil {
			// it's really hard to get this error cause the test fails before we read from the channel
			t.Logf("Error from run: %s", err.Error())
		}
		resultCh <- err
	}()

	// Wait for the server to start up
	require.Eventually(t, func() bool {
		res, err := http.Get(fmt.Sprintf("http://localhost:%d/v2/liveness", httpPort))
		if err != nil {
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
	case err := <-resultCh:
		// The Run function should exit without error when context is canceled
		require.NoError(t, err, "API server should shut down gracefully when context is canceled")
		t.Log("API server shut down successfully")
	case <-time.After(30 * time.Second):
		t.Fatal("API server failed to shut down within the expected time")
	}

	// Verify the server is no longer responding
	_, err := http.Get(fmt.Sprintf("http://localhost:%d/v2/liveness", httpPort))
	require.Error(t, err, "Server should no longer be responding after shutdown")
}
