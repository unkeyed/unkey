package vault

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/keys"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	return setupTestServiceWithClock(t, nil)
}

func setupTestServiceWithClock(t *testing.T, clk clock.Clock) *Service {
	t.Helper()
	// Use memory storage for fast, isolated tests
	memoryStorage, err := storage.NewMemory()
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	// Generate a random token for each test
	bearerToken := "test-token-" + uid.New("test")

	service, err := New(Config{
		Storage:     memoryStorage,
		MasterKey:   masterKey,
		BearerToken: bearerToken,
		Clock:       clk,
	})
	require.NoError(t, err)
	t.Cleanup(service.keyCache.Close)

	return service
}
