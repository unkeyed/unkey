package jwks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
)

func generateTestKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func serveJWKS(t *testing.T, keys map[string]*rsa.PublicKey) *httptest.Server {
	t.Helper()
	var jwkKeys []jose.JSONWebKey
	for kid, pub := range keys {
		jwkKeys = append(jwkKeys, jose.JSONWebKey{
			Key:       pub,
			KeyID:     kid,
			Algorithm: "RS256",
			Use:       "sig",
		})
	}

	keySet := jose.JSONWebKeySet{Keys: jwkKeys}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(keySet)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRemoteKeySet_GetKey(t *testing.T) {
	key1 := generateTestKeyPair(t)
	key2 := generateTestKeyPair(t)

	srv := serveJWKS(t, map[string]*rsa.PublicKey{
		"key-1": &key1.PublicKey,
		"key-2": &key2.PublicKey,
	})

	ks := NewRemoteKeySet(srv.URL)

	ctx := context.Background()

	// Should resolve both keys.
	got1, err := ks.GetKey(ctx, "key-1")
	require.NoError(t, err)
	require.True(t, key1.PublicKey.Equal(got1))

	got2, err := ks.GetKey(ctx, "key-2")
	require.NoError(t, err)
	require.True(t, key2.PublicKey.Equal(got2))
}

func TestRemoteKeySet_UnknownKey(t *testing.T) {
	key := generateTestKeyPair(t)
	srv := serveJWKS(t, map[string]*rsa.PublicKey{
		"key-1": &key.PublicKey,
	})

	ks := NewRemoteKeySet(srv.URL)

	ctx := context.Background()

	_, err := ks.GetKey(ctx, "nonexistent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRemoteKeySet_RefreshCooldown(t *testing.T) {
	key := generateTestKeyPair(t)
	fetchCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetchCount++
		keySet := jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{{
				Key:       &key.PublicKey,
				KeyID:     "key-1",
				Algorithm: "RS256",
				Use:       "sig",
			}},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(keySet)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	ks := NewRemoteKeySet(srv.URL, WithRefreshCooldown(time.Hour))

	ctx := context.Background()

	// First call triggers initial fetch.
	_, err := ks.GetKey(ctx, "key-1")
	require.NoError(t, err)
	require.Equal(t, 1, fetchCount)

	// Second call for unknown key should attempt refresh but be rate-limited.
	_, err = ks.GetKey(ctx, "nonexistent")
	require.Error(t, err)
	// Only 1 additional fetch because cooldown hasn't expired (initial + 1 refresh attempt that's blocked).
	// Actually, the cooldown check happens before the fetch, so the second call is blocked.
	require.Equal(t, 1, fetchCount)
}

func TestRemoteKeySet_CustomHTTPClient(t *testing.T) {
	key := generateTestKeyPair(t)
	srv := serveJWKS(t, map[string]*rsa.PublicKey{
		"key-1": &key.PublicKey,
	})

	customClient := &http.Client{Timeout: 5 * time.Second}
	ks := NewRemoteKeySet(srv.URL, WithHTTPClient(customClient))

	ctx := context.Background()

	got, err := ks.GetKey(ctx, "key-1")
	require.NoError(t, err)
	require.True(t, key.PublicKey.Equal(got))
}
