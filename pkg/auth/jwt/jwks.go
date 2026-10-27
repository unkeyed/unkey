package jwt

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
)

const jwksFetchTimeout = 10 * time.Second

const jwksMaxResponseBytes = 256 * 1024

// jwksRefetchMinInterval rate-limits JWKS fetches triggered by tokens naming
// an unknown signing key, so a storm of invalid tokens cannot hammer the JWKS
// endpoint. Key rotations are picked up within this interval.
const jwksRefetchMinInterval = time.Minute

// JWKSCacheKey isolates cached verifiers by their key source and claim constraints.
type JWKSCacheKey struct {
	URL      string
	Issuer   string
	Audience string
}

// NewJWKSCache creates a cache shared by JWT resolvers, with a five-minute
// hard expiry to bound trust in revoked signing keys. The caller must close it.
func NewJWKSCache(clk clock.Clock) (cache.Cache[JWKSCacheKey, *JWKS], error) {
	return cache.New(cache.Config[JWKSCacheKey, *JWKS]{
		Fresh:     5 * time.Minute,
		Stale:     5 * time.Minute,
		MaxSize:   100,
		Resource:  "jwks",
		Clock:     clk,
		Scheduler: nil,
	})
}

// jwksVerifier keeps fetch backoff separate from cache reads so a slow
// endpoint does not delay verification against unexpired keys.
type jwksVerifier struct {
	key   JWKSCacheKey
	keys  cache.Cache[JWKSCacheKey, *JWKS]
	clock clock.Clock

	fetchMu     sync.Mutex
	lastAttempt time.Time // guarded by fetchMu
	fetched     bool      // guarded by fetchMu
}

var _ claimsVerifier = (*jwksVerifier)(nil)

var jwksHTTPClient = &http.Client{
	Transport: jwksHTTPTransport(),
	Timeout:   jwksFetchTimeout,
}

func jwksHTTPTransport() http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ResponseHeaderTimeout = jwksFetchTimeout
	return transport
}

func (v *jwksVerifier) verify(ctx context.Context, token string) (Claims, error) {
	var zero Claims

	// Parsing the kid before touching the key source lets tokens with a
	// garbage header fail without costing a fetch or a signature check.
	kid, err := tokenKID(token)
	if err != nil {
		return zero, fault.Wrap(err,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal("failed to decode JWT header"),
			fault.Public("Invalid bearer token."),
		)
	}

	keys, err := v.load(ctx)
	if err != nil {
		return zero, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to fetch JWKS verification keys"),
			fault.Public("Unable to verify the bearer token right now."),
		)
	}

	claims, verifyErr := keys.verify(kid, token)
	if verifyErr == nil {
		return claims, nil
	}

	// A refetch can only help when the signing key may be absent from the
	// cached set: the token names an unknown kid, or names no kid at all. A
	// key the set already contains fails signature or claims validation
	// identically against a fresh fetch, so those rejections trigger nothing.
	if kid == "" || !keys.contains(kid) {
		if fresh, changed := v.refetch(ctx, keys); changed {
			if claims, retryErr := fresh.verify(kid, token); retryErr == nil {
				return claims, nil
			}
		}
	}

	return zero, fault.Wrap(verifyErr,
		fault.Code(codes.Auth.Authentication.Malformed.URN()),
		fault.Internal("failed to verify JWT"),
		fault.Public("Invalid bearer token."),
	)
}

// load returns an unexpired key set, fetching it on first use or expiry.
func (v *jwksVerifier) load(ctx context.Context) (*JWKS, error) {
	if set, hit := v.keys.Get(ctx, v.key); hit == cache.Hit {
		return set, nil
	}
	set, _, err := v.fetchLatest(ctx, nil)
	return set, err
}

// refetch refreshes the key set after a token failed against seen. It returns
// the freshest available set and whether that set differs from seen and is
// therefore worth retrying against.
func (v *jwksVerifier) refetch(ctx context.Context, seen *JWKS) (*JWKS, bool) {
	set, changed, err := v.fetchLatest(ctx, seen)
	if err != nil {
		return seen, false
	}
	return set, changed
}

// fetchLatest serializes JWKS fetches behind fetchMu. seen is the set the
// caller already failed against, or nil on a cache miss. It returns the freshest
// set, whether it differs from seen, and an error when no unexpired set is available.
//
// Only successful fetches write to the cache, so failed refreshes cannot renew
// trust in old keys. Cold-start failures do not arm backoff because no keys
// have been served yet.
func (v *jwksVerifier) fetchLatest(ctx context.Context, seen *JWKS) (*JWKS, bool, error) {
	v.fetchMu.Lock()
	defer v.fetchMu.Unlock()

	// A fetch may have completed while this goroutine waited for the lock.
	if current, hit := v.keys.Get(ctx, v.key); hit == cache.Hit && current != seen {
		return current, true, nil
	}

	// Once a key set has been served, throttle refetches so a token that failed
	// against a known key cannot hammer the endpoint. Before the first success
	// there is nothing to serve, so retry every request instead of wedging all
	// auth for the full interval after a transient cold-start failure.
	if v.fetched && v.clock.Now().Sub(v.lastAttempt) < jwksRefetchMinInterval {
		if current, hit := v.keys.Get(ctx, v.key); hit == cache.Hit {
			return current, current != seen, nil
		}
		return nil, false, errors.New("JWKS fetch failed recently, backing off")
	}
	v.lastAttempt = v.clock.Now()

	fetchCtx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()
	set, err := fetchKeySet(fetchCtx, v.key.Issuer, v.key.Audience, v.key.URL)
	if err != nil {
		if current, hit := v.keys.Get(ctx, v.key); hit == cache.Hit {
			return current, current != seen, nil
		}
		return nil, false, err
	}
	v.keys.Set(ctx, v.key, set)
	v.fetched = true
	return set, true, nil
}

// JWKS holds immutable RS256 verifiers from one successful JWKS fetch.
type JWKS struct {
	// byKID selects the verifier named by a token's kid header.
	byKID map[string]tokenjwt.Verifier[Claims]
	// ordered preserves document order for tokens that name no kid.
	ordered []tokenjwt.Verifier[Claims]
}

// contains reports whether the set has a verifier for the kid.
func (s *JWKS) contains(kid string) bool {
	_, ok := s.byKID[kid]
	return ok
}

// verify checks the token against the verifier its kid names, or against
// every key in document order when the token names no kid.
func (s *JWKS) verify(kid string, token string) (Claims, error) {
	var zero Claims

	if kid != "" {
		verifier, ok := s.byKID[kid]
		if !ok {
			return zero, fmt.Errorf("no JWKS key matches kid %q", kid)
		}
		return verifier.Verify(token)
	}

	var lastErr error
	for _, verifier := range s.ordered {
		claims, err := verifier.Verify(token)
		if err == nil {
			return claims, nil
		}
		lastErr = err
	}
	return zero, lastErr
}

// tokenKID returns the kid header naming the token's signing key, or "" when
// the token does not name one.
func tokenKID(token string) (string, error) {
	encodedHeader, _, _ := strings.Cut(token, ".")
	headerJSON, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return "", fmt.Errorf("decode JWT header: %w", err)
	}

	var header struct {
		KID string `json:"kid"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", fmt.Errorf("parse JWT header: %w", err)
	}
	return header.KID, nil
}

// jwksResponse is the JSON shape of a JWKS endpoint response.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// jwkKey is one JSON Web Key entry; only RS256 RSA signing keys are used.
type jwkKey struct {
	Algorithm string `json:"alg"`
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Modulus   string `json:"n"`
	Exponent  string `json:"e"`
	KeyID     string `json:"kid"`
}

// fetchKeySet downloads the JWKS document and indexes a verifier per usable
// RS256 signing key. A malformed entry is skipped rather than failing the
// fetch, so one bad key cannot invalidate the usable keys published next to
// it; only a document with no usable key at all is an error.
func fetchKeySet(ctx context.Context, issuer string, audience string, jwksURL string) (*JWKS, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := jwksHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch JWKS: unexpected status %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxResponseBytes)).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("JWKS contains no keys")
	}

	set := &JWKS{
		byKID:   map[string]tokenjwt.Verifier[Claims]{},
		ordered: nil,
	}
	var keyErrs []error
	for _, key := range jwks.Keys {
		verifier, err := verifierFromJWK(key, issuer, audience)
		if err != nil {
			keyErrs = append(keyErrs, err)
			continue
		}
		if verifier == nil {
			continue
		}
		set.ordered = append(set.ordered, verifier)
		if key.KeyID != "" {
			set.byKID[key.KeyID] = verifier
		}
	}
	if len(set.ordered) == 0 {
		return nil, errors.Join(errors.New("JWKS contains no usable RS256 signing keys"), errors.Join(keyErrs...))
	}

	return set, nil
}

// verifierFromJWK converts one JWK into an RS256 verifier, returning nil for
// keys with a different type, use, or algorithm so callers can skip them.
func verifierFromJWK(key jwkKey, issuer string, audience string) (tokenjwt.Verifier[Claims], error) {
	if key.KeyType != "RSA" {
		return nil, nil
	}
	if key.Use != "" && key.Use != "sig" {
		return nil, nil
	}
	if key.Algorithm != "" && key.Algorithm != "RS256" {
		return nil, nil
	}

	modulus, err := base64.RawURLEncoding.DecodeString(key.Modulus)
	if err != nil {
		return nil, fmt.Errorf("decode JWKS key %q modulus: %w", key.KeyID, err)
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(key.Exponent)
	if err != nil {
		return nil, fmt.Errorf("decode JWKS key %q exponent: %w", key.KeyID, err)
	}
	exponent := new(big.Int).SetBytes(exponentBytes)
	if !exponent.IsInt64() || exponent.Sign() <= 0 {
		return nil, fmt.Errorf("JWKS key %q has invalid exponent", key.KeyID)
	}

	publicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes(modulus),
		E: int(exponent.Int64()),
	}
	if publicKey.N.Sign() <= 0 {
		return nil, fmt.Errorf("JWKS key %q has invalid modulus", key.KeyID)
	}

	der, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("encode JWKS key %q public key: %w", key.KeyID, err)
	}
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:    "PUBLIC KEY",
		Headers: nil,
		Bytes:   der,
	}))

	return tokenjwt.NewRS256Verifier[Claims](publicKeyPEM, verifierOptions(issuer, audience)...)
}
