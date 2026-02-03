package dockertest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/dockertest"
)

func TestRestate(t *testing.T) {
	config := dockertest.Restate(t)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(config.AdminURL + "/health")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
