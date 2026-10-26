package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

const (
	fetchTimeout       = 5 * time.Second
	responseBytesMax   = 1 << 20
	refreshIntervalMin = time.Minute
	cacheLifetime      = 5 * time.Minute
)

// Config trusts one issuer and an exact set of service subjects, not workspace claims.
type Config struct {
	Issuer          string   `toml:"issuer"`
	Audience        string   `toml:"audience"`
	AllowedSubjects []string `toml:"allowed_subjects"`
}

func (c Config) Validate() error {
	if err := validateURL(c.Issuer); err != nil {
		return fmt.Errorf("oidc issuer: %w", err)
	}
	if strings.TrimSpace(c.Audience) == "" || len(c.AllowedSubjects) == 0 {
		return errors.New("oidc audience and allowed_subjects are required")
	}
	for _, subject := range c.AllowedSubjects {
		if strings.TrimSpace(subject) == "" || strings.ContainsAny(subject, "*?") {
			return errors.New("oidc allowed_subjects must contain nonempty exact subjects")
		}
	}
	return nil
}

type Verifier struct {
	config      Config
	client      *http.Client
	current     atomic.Pointer[keySet]
	mu          sync.Mutex
	attemptLast time.Time
}

type keySet struct {
	keys      map[string]jose.JSONWebKey
	fetchedAt time.Time
}

func New(cfg Config) (*Verifier, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	cfg.AllowedSubjects = slices.Clone(cfg.AllowedSubjects)
	return &Verifier{
		config: cfg,
		client: &http.Client{
			Timeout:       fetchTimeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("oidc redirects are not allowed") },
		},
		current:     atomic.Pointer[keySet]{},
		mu:          sync.Mutex{},
		attemptLast: time.Time{},
	}, nil
}

func (v *Verifier) Verify(ctx context.Context, raw string) error {
	if len(raw) == 0 || len(raw) > 16*1024 {
		return errors.New("invalid token size")
	}
	var claims jwt.RegisteredClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok || kid == "" || len(kid) > 256 {
			return nil, errors.New("missing or invalid signing key id")
		}
		set, err := v.keys(ctx, kid)
		if err != nil {
			return nil, err
		}
		key, ok := set.keys[kid]
		if !ok || (key.Algorithm != "" && key.Algorithm != token.Method.Alg()) {
			return nil, errors.New("unknown signing key")
		}
		return key.Key, nil
	}, jwt.WithValidMethods([]string{"RS256", "ES256"}), jwt.WithIssuer(v.config.Issuer), jwt.WithAudience(v.config.Audience), jwt.WithExpirationRequired())
	if err != nil {
		return fmt.Errorf("verify workload token: %w", err)
	}
	if !slices.Contains(v.config.AllowedSubjects, claims.Subject) {
		return errors.New("service subject is not allowed")
	}
	return nil
}

func (v *Verifier) keys(ctx context.Context, kid string) (*keySet, error) {
	if set := v.current.Load(); set != nil && time.Since(set.fetchedAt) < cacheLifetime {
		if _, ok := set.keys[kid]; ok {
			return set, nil
		}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	set := v.current.Load()
	if set != nil && time.Since(set.fetchedAt) < cacheLifetime {
		if _, ok := set.keys[kid]; ok {
			return set, nil
		}
	}
	if time.Since(v.attemptLast) < refreshIntervalMin {
		return nil, errors.New("signing key refresh is rate limited")
	}
	v.attemptLast = time.Now()
	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	var discovery struct {
		Issuer  string `json:"issuer"`
		JWKSURI string `json:"jwks_uri"`
	}
	if err := v.fetch(fetchCtx, strings.TrimSuffix(v.config.Issuer, "/")+"/.well-known/openid-configuration", &discovery); err != nil {
		return nil, err
	}
	if discovery.Issuer != v.config.Issuer {
		return nil, errors.New("discovery issuer mismatch")
	}
	var document jose.JSONWebKeySet
	if err := v.fetch(fetchCtx, discovery.JWKSURI, &document); err != nil {
		return nil, err
	}
	if len(document.Keys) == 0 || len(document.Keys) > 64 {
		return nil, errors.New("invalid signing key count")
	}
	keys := make(map[string]jose.JSONWebKey)
	for _, key := range document.Keys {
		if key.KeyID == "" || !key.Valid() || !key.IsPublic() || (key.Use != "" && key.Use != "sig") {
			continue
		}
		if _, exists := keys[key.KeyID]; exists {
			return nil, errors.New("duplicate signing key id")
		}
		keys[key.KeyID] = key
	}
	if len(keys) == 0 {
		return nil, errors.New("no usable signing keys")
	}
	set = &keySet{keys: keys, fetchedAt: time.Now()}
	v.current.Store(set)
	return set, nil
}

func (v *Verifier) fetch(ctx context.Context, address string, target any) (err error) {
	if err := validateURL(address); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, resp.Body.Close()) }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("oidc endpoint returned status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, responseBytesMax+1))
	if err != nil {
		return err
	}
	if len(data) > responseBytesMax {
		return errors.New("oidc response exceeds size limit")
	}
	return json.Unmarshal(data, target)
}

func validateURL(address string) error {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("expected an absolute HTTPS URL without credentials, query, or fragment")
	}
	return nil
}
