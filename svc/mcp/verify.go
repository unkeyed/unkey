package mcp

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
)

const (
	maxTokenBytes        = 32 * 1024
	jwksMaxResponseBytes = 256 * 1024
	jwksRefetchInterval  = time.Minute
)

var errKeysUnavailable = errors.New("authorization server keys are unavailable")

// Identity is the verified claim set returned by the whoami tool.
type Identity struct {
	Subject     string   `json:"sub"`
	OrgID       string   `json:"org_id"`
	ClientID    string   `json:"client_id"`
	SessionID   string   `json:"sid"`
	Scope       string   `json:"scope"`
	Audience    []string `json:"aud"`
	Issuer      string   `json:"iss"`
	ExpiresAt   int64    `json:"exp"`
	IssuedAt    int64    `json:"iat"`
	ExpMinusIAT int64    `json:"exp_minus_iat"`
}

type verifier struct {
	issuer    string
	jwksURL   string
	client    *http.Client
	logger    *slog.Logger
	logClaims bool
	now       func() time.Time

	mu        sync.Mutex
	keys      *keySet
	lastFetch time.Time
}

func newVerifier(cfg Config, logger *slog.Logger) *verifier {
	return &verifier{
		issuer:    cfg.Issuer,
		jwksURL:   cfg.JWKSURL,
		client:    jwksClient,
		logger:    logger,
		logClaims: cfg.LogClaims,
		now:       time.Now,
		mu:        sync.Mutex{},
		keys:      nil,
		lastFetch: time.Time{},
	}
}

func (v *verifier) tokenVerifier(resource string) sdkauth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*sdkauth.TokenInfo, error) {
		identity, err := v.verify(ctx, token, resource)
		if err != nil {
			return nil, err
		}
		// The middleware treats Expiration as exclusive. Adding one second
		// matches "expired only when now > exp" without accepting a later second.
		return &sdkauth.TokenInfo{
			Scopes:     strings.Fields(identity.Scope),
			Expiration: time.Unix(identity.ExpiresAt, 0).Add(time.Second),
			UserID:     identity.Subject,
			Extra:      map[string]any{"identity": identity},
		}, nil
	}
}

func (v *verifier) verify(ctx context.Context, token string, resource string) (Identity, error) {
	header, payload, signingInput, signature, err := splitToken(token)
	if err != nil {
		return zeroIdentity(), invalidToken("malformed token")
	}
	if header.Alg != "RS256" {
		return zeroIdentity(), invalidToken("malformed token")
	}

	keys, err := v.keysFor(ctx, header.Kid)
	if err != nil {
		v.logger.Error("mcp spike jwks fetch failed", "error", err.Error())
		return zeroIdentity(), errKeysUnavailable
	}
	if !verifyRS256(keys, header.Kid, signingInput, signature) {
		return zeroIdentity(), invalidToken("invalid signature")
	}

	raw, err := decodeClaimMap(payload)
	if err != nil {
		return zeroIdentity(), invalidToken("malformed token")
	}
	// The signature is already checked, so the payload is from AuthKit.
	// Logging before the aud comparison is what makes a mismatched staging
	// token visible. signingInput and signature are not logged.
	if v.logClaims {
		v.logger.Info("mcp spike decoded jwt claims",
			"resource", resource,
			"claim_names", slices.Sorted(maps.Keys(raw)),
			"claims", raw,
		)
	}

	claims, err := decodeClaims(payload)
	if err != nil {
		return zeroIdentity(), invalidToken("malformed token")
	}
	if reason := claims.rejection(v.issuer, resource, v.now()); reason != "" {
		if v.logClaims {
			v.logger.Info("mcp spike rejected jwt", "resource", resource, "reason", reason)
		}
		return zeroIdentity(), invalidToken(reason)
	}
	return claims.identity(), nil
}

func (v *verifier) keysFor(ctx context.Context, kid string) (*keySet, error) {
	v.mu.Lock()
	cached := v.keys
	v.mu.Unlock()
	if cached != nil && (kid == "" || cached.has(kid)) {
		return cached, nil
	}
	return v.refresh(ctx)
}

func (v *verifier) refresh(ctx context.Context) (*keySet, error) {
	v.mu.Lock()
	if v.keys != nil && time.Since(v.lastFetch) < jwksRefetchInterval {
		cached := v.keys
		v.mu.Unlock()
		return cached, nil
	}
	v.mu.Unlock()

	fetched, err := v.fetch(ctx)
	if err != nil {
		return nil, err
	}
	v.mu.Lock()
	v.keys = fetched
	v.lastFetch = time.Now()
	v.mu.Unlock()
	return fetched, nil
}

func (v *verifier) fetch(ctx context.Context) (*keySet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("jwks status %d", resp.StatusCode)
	}

	var document jwksDocument
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxResponseBytes)).Decode(&document); err != nil {
		return nil, err
	}

	published := make([]publishedKey, 0, len(document.Keys))
	set := &keySet{byID: map[string]*rsa.PublicKey{}, all: nil}
	for _, key := range document.Keys {
		published = append(published, publishedKey{
			Kid: key.Kid,
			Kty: key.Kty,
			Alg: key.Alg,
			Use: key.Use,
		})
		pub, ok := key.rsaPublicKey()
		if !ok {
			continue
		}
		set.all = append(set.all, pub)
		if key.Kid != "" {
			set.byID[key.Kid] = pub
		}
	}
	v.logger.Info("mcp spike jwks keys", "count", len(published), "keys", published)
	if len(set.all) == 0 {
		return nil, errors.New("jwks contains no RS256 signing keys")
	}
	return set, nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type connectClaims struct {
	Issuer    string   `json:"iss"`
	Subject   string   `json:"sub"`
	Audience  audience `json:"aud"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	ClientID  string   `json:"client_id"`
	OrgID     string   `json:"org_id"`
	SessionID string   `json:"sid"`
	Scope     string   `json:"scope"`
}

func (c connectClaims) rejection(issuer string, resource string, now time.Time) string {
	if c.Issuer != issuer {
		return "issuer mismatch"
	}
	if !c.Audience.matches(resource) {
		return "audience mismatch"
	}
	if c.ExpiresAt <= 0 {
		return "exp is required"
	}
	if now.Unix() > c.ExpiresAt {
		return "token expired"
	}
	if c.OrgID == "" {
		return "org_id is required"
	}
	return ""
}

func (c connectClaims) identity() Identity {
	audience := c.Audience.values
	if audience == nil {
		audience = []string{}
	}
	return Identity{
		Subject:     c.Subject,
		OrgID:       c.OrgID,
		ClientID:    c.ClientID,
		SessionID:   c.SessionID,
		Scope:       c.Scope,
		Audience:    audience,
		Issuer:      c.Issuer,
		ExpiresAt:   c.ExpiresAt,
		IssuedAt:    c.IssuedAt,
		ExpMinusIAT: c.ExpiresAt - c.IssuedAt,
	}
}

type audience struct {
	values []string
}

func (a *audience) UnmarshalJSON(data []byte) error {
	data = bytesTrim(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		a.values = []string{value}
		return nil
	}
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	a.values = values
	return nil
}

func (a audience) matches(resource string) bool {
	for _, value := range a.values {
		if value == resource {
			return true
		}
	}
	return false
}

func splitToken(token string) (jwtHeader, []byte, []byte, []byte, error) {
	var header jwtHeader
	if token == "" || len(token) > maxTokenBytes || strings.Count(token, ".") != 2 {
		return header, nil, nil, nil, errors.New("malformed token")
	}
	parts := strings.Split(token, ".")
	if parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return header, nil, nil, nil, errors.New("malformed token")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return header, nil, nil, nil, err
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return header, nil, nil, nil, err
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return header, nil, nil, nil, err
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return header, nil, nil, nil, err
	}
	return header, payload, []byte(parts[0] + "." + parts[1]), signature, nil
}

func decodeClaimMap(payload []byte) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return nil, err
	}
	if raw == nil {
		return map[string]any{}, nil
	}
	return raw, nil
}

func decodeClaims(payload []byte) (connectClaims, error) {
	var claims connectClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return connectClaims{
			Issuer:    "",
			Subject:   "",
			Audience:  audience{values: nil},
			ExpiresAt: 0,
			IssuedAt:  0,
			ClientID:  "",
			OrgID:     "",
			SessionID: "",
			Scope:     "",
		}, err
	}
	return claims, nil
}

func verifyRS256(set *keySet, kid string, signingInput []byte, signature []byte) bool {
	candidates := set.all
	if kid != "" {
		key, ok := set.byID[kid]
		if !ok {
			return false
		}
		candidates = []*rsa.PublicKey{key}
	}
	sum := sha256.Sum256(signingInput)
	for _, key := range candidates {
		if rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], signature) == nil {
			return true
		}
	}
	return false
}

func invalidToken(reason string) error {
	return fmt.Errorf("%w: %s", sdkauth.ErrInvalidToken, reason)
}

func zeroIdentity() Identity {
	return Identity{
		Subject:     "",
		OrgID:       "",
		ClientID:    "",
		SessionID:   "",
		Scope:       "",
		Audience:    nil,
		Issuer:      "",
		ExpiresAt:   0,
		IssuedAt:    0,
		ExpMinusIAT: 0,
	}
}

type keySet struct {
	byID map[string]*rsa.PublicKey
	all  []*rsa.PublicKey
}

func (s *keySet) has(kid string) bool {
	_, ok := s.byID[kid]
	return ok
}

type jwksDocument struct {
	Keys []jwk `json:"keys"`
}

type publishedKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (k jwk) rsaPublicKey() (*rsa.PublicKey, bool) {
	if k.Kty != "RSA" {
		return nil, false
	}
	if k.Use != "" && k.Use != "sig" {
		return nil, false
	}
	if k.Alg != "" && k.Alg != "RS256" {
		return nil, false
	}
	modulus, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, false
	}
	exponentBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, false
	}
	exponent := new(big.Int).SetBytes(exponentBytes)
	if !exponent.IsInt64() || exponent.Sign() <= 0 {
		return nil, false
	}
	pub := &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: int(exponent.Int64())}
	if pub.N.Sign() <= 0 {
		return nil, false
	}
	return pub, true
}

func bytesTrim(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}

var jwksClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}
