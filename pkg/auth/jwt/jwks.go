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
	"math/big"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	tokenjwt "github.com/unkeyed/unkey/pkg/jwt"
)

const jwksFetchTimeout = 10 * time.Second

// jwksRefetchMinInterval rate-limits JWKS fetches triggered by tokens naming
// an unknown signing key, so a storm of invalid tokens cannot hammer the JWKS
// endpoint. Key rotations are picked up within this interval.
const jwksRefetchMinInterval = time.Minute

// jwksVerifier verifies RS256 tokens with signing keys fetched from a JWKS
// endpoint. The key set is fetched lazily on first use and refetched when a
// token's signing key may be missing from the cached set, so signing-key
// rotations are picked up without a restart.
type jwksVerifier struct {
	jwksURL  string
	issuer   string
	audience string
	clock    clock.Clock

	// current is read lock-free on every verification. fetchMu serializes
	// only the HTTP fetch and its backoff bookkeeping, so a slow JWKS
	// endpoint (up to jwksFetchTimeout) never delays requests whose tokens
	// verify against the cached key set. Only cold-start callers, which have
	// no keys to serve anyway, and refetch callers, whose token already
	// failed against the cache, ever wait on it.
	current     atomic.Pointer[keySet]
	fetchMu     sync.Mutex
	lastAttempt time.Time // guarded by fetchMu
}

var _ claimsVerifier = (*jwksVerifier)(nil)

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

// load returns the current key set, fetching it on first use.
func (v *jwksVerifier) load(ctx context.Context) (*keySet, error) {
	if set := v.current.Load(); set != nil {
		return set, nil
	}
	set, _, err := v.fetchLatest(ctx, nil)
	return set, err
}

// refetch refreshes the key set after a token failed against seen. It returns
// the freshest available set and whether that set differs from seen and is
// therefore worth retrying against.
func (v *jwksVerifier) refetch(ctx context.Context, seen *keySet) (*keySet, bool) {
	set, changed, err := v.fetchLatest(ctx, seen)
	if err != nil {
		return seen, false
	}
	return set, changed
}

// fetchLatest serializes JWKS fetches behind fetchMu. seen is the set the
// caller already failed against, or nil on first use. It returns the freshest
// set, whether that set differs from seen, and an error only when no set is
// available at all.
//
// Fetch attempts are rate-limited to one per jwksRefetchMinInterval once a key
// set has been served; inside the window callers keep the set they have. A
// failed fetch keeps serving the previous set, so a JWKS endpoint outage does
// not invalidate tokens signed by already known keys. Before the first
// successful fetch the rate limit is not armed, so a transient cold-start
// failure does not wedge all auth for the full interval.
func (v *jwksVerifier) fetchLatest(ctx context.Context, seen *keySet) (*keySet, bool, error) {
	v.fetchMu.Lock()
	defer v.fetchMu.Unlock()

	// A fetch may have completed while this goroutine waited for the lock.
	if current := v.current.Load(); current != nil && current != seen {
		return current, true, nil
	}

	// Once a key set has been served, throttle refetches so a token that failed
	// against a known key cannot hammer the endpoint. Before the first success
	// there is nothing to serve, so retry every request instead of wedging all
	// auth for the full interval after a transient cold-start failure.
	if v.current.Load() != nil && !v.lastAttempt.IsZero() && v.clock.Now().Sub(v.lastAttempt) < jwksRefetchMinInterval {
		if seen != nil {
			return seen, false, nil
		}
		return nil, false, errors.New("JWKS fetch failed recently, backing off")
	}
	v.lastAttempt = v.clock.Now()

	fetchCtx, cancel := context.WithTimeout(ctx, jwksFetchTimeout)
	defer cancel()
	set, err := fetchKeySet(fetchCtx, v.issuer, v.audience, v.jwksURL)
	if err != nil {
		if seen != nil {
			return seen, false, nil
		}
		return nil, false, err
	}
	v.current.Store(set)
	return set, true, nil
}

// keySet is one immutable JWKS fetch result; the verifier swaps whole sets
// atomically and never mutates one in place.
type keySet struct {
	// byKID selects the verifier named by a token's kid header.
	byKID map[string]tokenjwt.Verifier[Claims]
	// ordered preserves document order for tokens that name no kid.
	ordered []tokenjwt.Verifier[Claims]
}

// contains reports whether the set has a verifier for the kid.
func (s *keySet) contains(kid string) bool {
	_, ok := s.byKID[kid]
	return ok
}

// verify checks the token against the verifier its kid names, or against
// every key in document order when the token names no kid.
func (s *keySet) verify(kid string, token string) (Claims, error) {
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
func fetchKeySet(ctx context.Context, issuer string, audience string, jwksURL string) (*keySet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
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
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}
	if len(jwks.Keys) == 0 {
		return nil, errors.New("JWKS contains no keys")
	}

	set := &keySet{
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
