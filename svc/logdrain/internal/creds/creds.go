// Package creds caches decrypted log-drain credentials so the hot path
// does not hit svc/vault on every batch. Vault is the single decryption
// boundary; the cache holds the plaintext only inside this process.
//
// Cache invalidation is by drain ID: when a credential rotates, the
// coordinator calls Invalidate(drainID) so the next Get triggers a fresh
// Decrypt RPC. Storage uses pkg/cache with configurable TTL for security,
// providing metrics, LRU eviction, and SWR semantics.
package creds

import (
	"context"
	"errors"
	"fmt"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	vaultrpc "github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/defaults"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
)

// Config tunes the underlying pkg/cache. Defaults are fine in production —
// credentials almost never change, so a generous Stale window keeps hit
// rate near 100% with no observable cost.
type Config struct {
	// MaxSize caps the number of distinct drains held in memory. Eviction
	// is LRU; an evicted entry costs one extra Vault round-trip on the
	// next access.
	MaxSize int
	// Fresh and Stale govern when a cached entry must be re-fetched. We
	// keep both long because credentials are mutated explicitly via
	// Invalidate, not by elapsed time.
	Fresh time.Duration
	Stale time.Duration
	Clock clock.Clock
}

// Cache decrypts log-drain ciphertexts via Vault and memoises the plaintext
// keyed by drain ID. Concurrent-safe.
//
// Tests use svc/vault/testutil.StartTestVaultWithMemory to spin up a real
// Vault against in-memory storage; no fake-interface is needed because
// vaultrpc.VaultServiceClient is already an interface and the test util
// returns a connect client that wraps cleanly into it via
// vaultrpc.NewConnectVaultServiceClient.
type Cache struct {
	vault vaultrpc.VaultServiceClient
	cache cache.Cache[string, string]
}

// NewCache builds a Cache. cfg may be the zero value for sensible defaults.
func NewCache(client vaultrpc.VaultServiceClient, cfg Config) (*Cache, error) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    defaults.Or(cfg.Fresh, 24*time.Hour),
		Stale:    defaults.Or(cfg.Stale, 7*24*time.Hour),
		MaxSize:  defaults.Or(cfg.MaxSize, 10_000),
		Resource: "logdrain_creds",
		// clock.New returns *RealClock; the lambda widens it to the
		// interface type so generic inference unifies with cfg.Clock.
		Clock: defaults.OrFunc(cfg.Clock, func() clock.Clock { return clock.New() }),
	})
	if err != nil {
		return nil, fmt.Errorf("creds: build cache: %w", err)
	}

	return &Cache{vault: client, cache: c}, nil
}

// Lookup describes the row that Get decrypts. The keyring is the workspace
// ID per the convention used elsewhere (see svc/api/routes/v2_keys_reroll
// and svc/ctrl/services/acme); rotating workspace-level KMS material then
// re-encrypts every drain credential in one sweep.
type Lookup struct {
	DrainID     string
	WorkspaceID string
	Ciphertext  string
}

// ErrEmptyCiphertext is returned when a Lookup carries no ciphertext, e.g.
// for OAuth-backed drains whose credential lives on oauth_grants. The
// coordinator should resolve those via a separate path before calling Get.
var ErrEmptyCiphertext = errors.New("creds: empty ciphertext (use the OAuth grant path instead)")

// Get returns the plaintext credential for the given drain. The first call
// hits Vault; subsequent calls within the cache's Stale window return the
// memoised value until Invalidate is called.
func (c *Cache) Get(ctx context.Context, l Lookup) (string, error) {
	if l.Ciphertext == "" {
		return "", ErrEmptyCiphertext
	}

	if v, hit := c.cache.Get(ctx, l.DrainID); hit == cache.Hit {
		metrics.CredentialCacheHits.WithLabelValues("hit").Inc()
		return v, nil
	}

	metrics.CredentialCacheHits.WithLabelValues("miss").Inc()

	resp, err := c.vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   l.WorkspaceID,
		Encrypted: l.Ciphertext,
	})
	if err != nil {
		metrics.CredentialDecrypts.WithLabelValues("vault_error").Inc()
		return "", fmt.Errorf("vault decrypt drain=%s: %w", l.DrainID, err)
	}

	metrics.CredentialDecrypts.WithLabelValues("success").Inc()
	c.cache.Set(ctx, l.DrainID, resp.Plaintext)
	return resp.Plaintext, nil
}

// Invalidate drops the cached plaintext for one drain. Called when the
// dashboard rotates a paste-token credential or when an OAuth grant is
// re-bound; the next Get re-fetches from Vault.
func (c *Cache) Invalidate(ctx context.Context, drainID string) {
	c.cache.Remove(ctx, drainID)
}
