package creds_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	vaultrpc "github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/svc/logdrain/internal/creds"
	"github.com/unkeyed/unkey/svc/vault/testutil"
)

// startVault spins up a real in-memory vault and returns a wrapped client
// plus a helper that encrypts a string for a given workspace keyring.
// Using the real service rather than a fake catches API or keyring drift
// the same way an integration test would, without needing dockertest.
func startVault(t *testing.T) (vaultrpc.VaultServiceClient, func(workspaceID, plaintext string) string) {
	t.Helper()
	tv := testutil.StartTestVaultWithMemory(t)
	client := vaultrpc.NewConnectVaultServiceClient(tv.Client)
	encrypt := func(workspaceID, plaintext string) string {
		enc, err := client.Encrypt(context.Background(), &vaultv1.EncryptRequest{
			Keyring: workspaceID,
			Data:    plaintext,
		})
		require.NoError(t, err)
		return enc.Encrypted
	}
	return client, encrypt
}

func TestCache_Get_RoundTripsThroughVault(t *testing.T) {
	t.Parallel()

	client, encrypt := startVault(t)
	c, err := creds.NewCache(client, creds.Config{})
	require.NoError(t, err)

	got, err := c.Get(context.Background(), creds.Lookup{
		DrainID:     "ld_x",
		WorkspaceID: "ws_1",
		Ciphertext:  encrypt("ws_1", "plain-token"),
	})
	require.NoError(t, err)
	require.Equal(t, "plain-token", got)
}

// countingClient wraps a real vault client and counts Decrypt calls. Used
// to prove the cache hits on the second Get and that Invalidate forces a
// refetch — facts the in-process cache must guarantee but the real Vault
// service alone cannot demonstrate.
type countingClient struct {
	inner vaultrpc.VaultServiceClient
	calls atomic.Int64
}

func (c *countingClient) Liveness(ctx context.Context, req *vaultv1.LivenessRequest) (*vaultv1.LivenessResponse, error) {
	return c.inner.Liveness(ctx, req)
}
func (c *countingClient) Encrypt(ctx context.Context, req *vaultv1.EncryptRequest) (*vaultv1.EncryptResponse, error) {
	return c.inner.Encrypt(ctx, req)
}
func (c *countingClient) Decrypt(ctx context.Context, req *vaultv1.DecryptRequest) (*vaultv1.DecryptResponse, error) {
	c.calls.Add(1)
	return c.inner.Decrypt(ctx, req)
}
func (c *countingClient) EncryptBulk(ctx context.Context, req *vaultv1.EncryptBulkRequest) (*vaultv1.EncryptBulkResponse, error) {
	return c.inner.EncryptBulk(ctx, req)
}
func (c *countingClient) DecryptBulk(ctx context.Context, req *vaultv1.DecryptBulkRequest) (*vaultv1.DecryptBulkResponse, error) {
	return c.inner.DecryptBulk(ctx, req)
}
func (c *countingClient) ReEncrypt(ctx context.Context, req *vaultv1.ReEncryptRequest) (*vaultv1.ReEncryptResponse, error) {
	return c.inner.ReEncrypt(ctx, req)
}

func TestCache_Get_HitOnSecondCall(t *testing.T) {
	t.Parallel()

	inner, encrypt := startVault(t)
	counted := &countingClient{inner: inner}
	c, err := creds.NewCache(counted, creds.Config{})
	require.NoError(t, err)

	l := creds.Lookup{DrainID: "ld_x", WorkspaceID: "ws_1", Ciphertext: encrypt("ws_1", "plain-token")}

	_, err = c.Get(context.Background(), l)
	require.NoError(t, err)
	_, err = c.Get(context.Background(), l)
	require.NoError(t, err)
	// Second call must not re-hit Vault — caching is the whole reason this
	// package exists; a regression here means re-decrypting on every batch.
	require.Equal(t, int64(1), counted.calls.Load())
}

func TestCache_Invalidate_ForcesRefetch(t *testing.T) {
	t.Parallel()

	inner, encrypt := startVault(t)
	counted := &countingClient{inner: inner}
	c, err := creds.NewCache(counted, creds.Config{})
	require.NoError(t, err)

	l := creds.Lookup{DrainID: "ld_x", WorkspaceID: "ws_1", Ciphertext: encrypt("ws_1", "plain")}

	_, err = c.Get(context.Background(), l)
	require.NoError(t, err)
	c.Invalidate(context.Background(), "ld_x")
	_, err = c.Get(context.Background(), l)
	require.NoError(t, err)
	require.Equal(t, int64(2), counted.calls.Load())
}

func TestCache_Get_EmptyCiphertextErrors(t *testing.T) {
	t.Parallel()

	// OAuth-backed drains carry no ciphertext on log_drain_credentials
	// (the credential lives on oauth_grants). The cache must reject them
	// loudly so the coordinator routes them through the OAuth path
	// instead of silently returning "".
	client, _ := startVault(t)
	c, err := creds.NewCache(client, creds.Config{})
	require.NoError(t, err)
	_, err = c.Get(context.Background(), creds.Lookup{DrainID: "ld", WorkspaceID: "ws"})
	require.ErrorIs(t, err, creds.ErrEmptyCiphertext)
}
