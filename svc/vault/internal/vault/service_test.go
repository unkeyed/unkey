package vault

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
)

func setupTestService(t *testing.T) *Service {
	logger := logging.NewNoop()

	// Use memory storage for fast, isolated tests
	memoryStorage, err := storage.NewMemory(storage.MemoryConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	// Generate a random token for each test
	bearerToken := "test-token-" + uid.New("test")

	service, err := New(Config{
		Logger:      logger,
		Storage:     memoryStorage,
		MasterKeys:  []string{masterKey},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	return service
}
