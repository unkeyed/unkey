package jwks

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// KeySet resolves public keys by key ID for JWT verification.
type KeySet interface {
	// GetKey returns the RSA public key matching the given key ID.
	// Returns an error if the key is not found or cannot be fetched.
	GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error)
}

// RemoteKeySet fetches and caches public keys from a JWKS endpoint.
// It is safe for concurrent use.
//
// Keys are fetched lazily on first access and cached in memory. When a
// requested kid is not found in the cache, a refresh is attempted (rate-limited
// by a cooldown period to prevent abuse).
type RemoteKeySet struct {
	jwksURL         string
	client          *http.Client
	refreshCooldown time.Duration

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	lastRefresh time.Time
	fetched     bool
}

// Option configures a RemoteKeySet.
type Option func(*RemoteKeySet)

// WithHTTPClient sets the HTTP client used to fetch the JWKS endpoint.
func WithHTTPClient(client *http.Client) Option {
	return func(r *RemoteKeySet) {
		r.client = client
	}
}

// WithRefreshCooldown sets the minimum interval between JWKS refresh attempts.
// Defaults to 1 minute.
func WithRefreshCooldown(d time.Duration) Option {
	return func(r *RemoteKeySet) {
		r.refreshCooldown = d
	}
}

// NewRemoteKeySet creates a RemoteKeySet that fetches public keys from the
// given JWKS URL. Keys are not fetched until the first call to GetKey.
func NewRemoteKeySet(jwksURL string, opts ...Option) *RemoteKeySet {
	r := &RemoteKeySet{
		jwksURL:         jwksURL,
		client:          http.DefaultClient,
		refreshCooldown: time.Minute,
		keys:            make(map[string]*rsa.PublicKey),
		mu:              sync.RWMutex{},
		lastRefresh:     time.Time{},
		fetched:         false,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// GetKey returns the RSA public key for the given key ID. If the key is not
// in the cache, it attempts to refresh from the JWKS endpoint (subject to
// the refresh cooldown).
func (r *RemoteKeySet) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Fast path: check cache under read lock.
	r.mu.RLock()
	if key, ok := r.keys[kid]; ok {
		r.mu.RUnlock()
		return key, nil
	}
	r.mu.RUnlock()

	// Slow path: refresh and retry.
	if err := r.refresh(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh JWKS: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	key, ok := r.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

// refresh fetches the JWKS endpoint and updates the key cache. It respects
// the cooldown period to prevent excessive requests. If the cache has never
// been populated, the cooldown is ignored.
func (r *RemoteKeySet) refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Respect cooldown unless this is the initial fetch.
	if r.fetched && time.Since(r.lastRefresh) < r.refreshCooldown {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB limit
	if err != nil {
		return fmt.Errorf("reading JWKS response: %w", err)
	}

	var raw rawJWKS
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("parsing JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(raw.Keys))
	for _, jwk := range raw.Keys {
		if jwk.Kty != "RSA" || jwk.Kid == "" {
			continue
		}
		pub, err := jwk.rsaPublicKey()
		if err != nil {
			continue
		}
		keys[jwk.Kid] = pub
	}

	r.keys = keys
	r.lastRefresh = time.Now()
	r.fetched = true
	return nil
}

// rawJWKS is a minimal JWKS representation that only extracts the fields we
// need. This avoids go-jose's strict x5c certificate parsing, which rejects
// certificates with negative serial numbers, this happened with WorkOS... So Yeet?.
type rawJWKS struct {
	Keys []rawJWK `json:"keys"`
}

type rawJWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (k *rawJWK) rsaPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decoding modulus: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decoding exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
