// Package testutil provides test utilities for creating a real vault service.
// This package lives in svc/vault/ so it can import internal packages,
// but exposes a public API for other packages to use in integration tests.
package testutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
)

// TestVault holds the vault service and client for testing.
type TestVault struct {
	// URL is the HTTP server URL for the vault service.
	URL string
	// Token is the bearer token for authenticating requests.
	Token string
	// Client is a pre-configured vault client with auth interceptor.
	Client vaultv1connect.VaultServiceClient
}

// StartTestVault creates an in-memory vault with S3 storage and HTTP server.
// Returns a TestVault with a VaultServiceClient configured to connect to it.
// The vault uses MinIO for S3-compatible storage and generates a fresh master key.
// All resources are cleaned up when the test completes.
func StartTestVault(t *testing.T) *TestVault {
	t.Helper()

	// Start S3 for vault storage
	s3 := dockertest.S3(t)

	// Create S3 storage
	st, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          "vault-test",
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.SecretAccessKey,
	})
	require.NoError(t, err)

	// Generate master key
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	token := "test-vault-token"

	// Create vault service
	v, err := vault.New(vault.Config{
		Storage:     st,
		MasterKeys:  []string{masterKey},
		BearerToken: token,
	})
	require.NoError(t, err)

	// Start HTTP server
	mux := http.NewServeMux()
	mux.Handle(vaultv1connect.NewVaultServiceHandler(v))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Create client with auth interceptor
	client := vaultv1connect.NewVaultServiceClient(
		server.Client(),
		server.URL,
		connect.WithInterceptors(authInterceptor(token)),
	)

	return &TestVault{
		URL:    server.URL,
		Token:  token,
		Client: client,
	}
}

// StartTestVaultWithMemory creates a vault with in-memory storage (no S3).
// This is faster for tests that don't need persistent storage.
func StartTestVaultWithMemory(t *testing.T) *TestVault {
	t.Helper()

	// Create in-memory storage
	st, err := storage.NewMemory()
	require.NoError(t, err)

	// Generate master key
	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	token := "test-vault-token"

	// Create vault service
	v, err := vault.New(vault.Config{
		Storage:     st,
		MasterKeys:  []string{masterKey},
		BearerToken: token,
	})
	require.NoError(t, err)

	// Start HTTP server
	mux := http.NewServeMux()
	mux.Handle(vaultv1connect.NewVaultServiceHandler(v))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	// Create client with auth interceptor
	client := vaultv1connect.NewVaultServiceClient(
		server.Client(),
		server.URL,
		connect.WithInterceptors(authInterceptor(token)),
	)

	return &TestVault{
		URL:    server.URL,
		Token:  token,
		Client: client,
	}
}

// authInterceptor adds the bearer token to all outgoing requests.
func authInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}
