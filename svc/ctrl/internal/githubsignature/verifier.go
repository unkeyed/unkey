package githubsignature

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
)

const (
	keysTTL          = time.Hour
	retryDelay       = time.Minute
	responseBytesMax = 1 << 20
)

// Verifier is safe for concurrent use and must be constructed with New.
type Verifier struct {
	keysURL     string
	clock       clock.Clock
	client      *http.Client
	mu          sync.Mutex
	keys        map[string]*ecdsa.PublicKey
	refreshAt   time.Time
	refreshDone chan struct{}
	refreshErr  error
}

// New caches signing keys for one hour and limits failed refreshes to once per
// minute. The supplied clock must be non-nil.
func New(keysURL string, clock clock.Clock) *Verifier {
	return &Verifier{
		keysURL:     keysURL,
		clock:       clock,
		client:      &http.Client{Timeout: 10 * time.Second},
		mu:          sync.Mutex{},
		keys:        nil,
		refreshAt:   time.Time{},
		refreshDone: nil,
		refreshErr:  nil,
	}
}

// Verify authenticates exact payload bytes. Invalid signatures and unknown keys
// return false without an error; retrieval failures use stale keys when available.
func (v *Verifier) Verify(ctx context.Context, payload []byte, signature, keyID string) (bool, error) {
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || len(signatureBytes) == 0 || keyID == "" {
		return false, nil
	}
	keys, err := v.publicKeys(ctx)
	if err != nil {
		return false, err
	}
	key, ok := keys[keyID]
	if !ok {
		return false, nil
	}
	digest := sha256.Sum256(payload)
	return ecdsa.VerifyASN1(key, digest[:], signatureBytes), nil
}

func (v *Verifier) publicKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	for {
		v.mu.Lock()
		if v.clock.Now().Before(v.refreshAt) {
			keys, err := v.keys, v.refreshErr
			v.mu.Unlock()
			if keys != nil {
				return keys, nil
			}
			return nil, err
		}
		if done := v.refreshDone; done != nil {
			v.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-done:
				continue
			}
		}
		v.refreshDone = make(chan struct{})
		v.mu.Unlock()

		keys, err := v.fetchKeys(ctx)
		v.mu.Lock()
		v.refreshErr = err
		v.refreshAt = v.clock.Now().Add(retryDelay)
		if err == nil {
			v.keys = keys
			v.refreshAt = v.clock.Now().Add(keysTTL)
		}
		close(v.refreshDone)
		v.refreshDone = nil
		v.mu.Unlock()
	}
}

func (v *Verifier) fetchKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return nil, errors.New("invalid github keys URL")
	}
	req.Header.Set("User-Agent", "unkey-secret-scanning")
	res, err := v.client.Do(req)
	if err != nil {
		return nil, errors.New("github signing keys unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, errors.New("github signing keys returned unsuccessful status")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, responseBytesMax+1))
	if err != nil || len(body) > responseBytesMax {
		return nil, errors.New("cannot read github signing keys response")
	}
	var response struct {
		PublicKeys []struct {
			KeyIdentifier string `json:"key_identifier"`
			Key           string `json:"key"`
		} `json:"public_keys"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.New("invalid github signing keys response")
	}
	keys := make(map[string]*ecdsa.PublicKey)
	for _, entry := range response.PublicKeys {
		block, _ := pem.Decode([]byte(entry.Key))
		if block == nil || block.Type != "PUBLIC KEY" || entry.KeyIdentifier == "" {
			continue
		}
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			continue
		}
		key, ok := parsed.(*ecdsa.PublicKey)
		if ok && key.Curve == elliptic.P256() {
			keys[entry.KeyIdentifier] = key
		}
	}
	if len(keys) == 0 {
		return nil, errors.New("github response contains no usable signing keys")
	}
	return keys, nil
}
