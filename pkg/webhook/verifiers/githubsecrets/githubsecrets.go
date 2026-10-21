package githubsecrets

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
	"fmt"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/singleflight"
	"github.com/unkeyed/unkey/pkg/webhook"
)

// EventType identifies secret-scanning reports, which carry no event-type header.
const EventType = "secret_scanning"

const responseBytesMax = 1 << 20

// Verifier is safe for concurrent use and must be constructed with New.
type Verifier struct {
	keysURL      string
	client       *http.Client
	keys         cache.Cache[string, map[string]*ecdsa.PublicKey]
	fetchResults cache.Cache[string, keyFetchResult]
	fetchGroup   singleflight.Group[keyFetchResult]
}

var _ webhook.Verifier = (*Verifier)(nil)

// Verify authenticates the raw body with GitHub's public key. Key retrieval
// failures use stale keys when available, or return webhook.ErrUnavailable.
func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {
	signature := r.Header.Get("Github-Public-Key-Signature")
	keyID := r.Header.Get("Github-Public-Key-Identifier")
	if signature == "" || keyID == "" {
		return webhook.Event{}, fmt.Errorf("%w: missing signing headers", webhook.ErrBadRequest)
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return webhook.Event{}, err
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || len(signatureBytes) == 0 {
		return webhook.Event{}, errors.New("invalid github signature encoding")
	}

	keys, err := v.publicKeys(r.Context(), keyID)
	if err != nil {
		return webhook.Event{}, fmt.Errorf("%w: %w", webhook.ErrUnavailable, err)
	}
	key, ok := keys[keyID]
	if !ok {
		return webhook.Event{}, errors.New("unknown github signing key")
	}

	digest := sha256.Sum256(payload)
	if !ecdsa.VerifyASN1(key, digest[:], signatureBytes) {
		return webhook.Event{}, errors.New("github signature mismatch")
	}
	return webhook.Event{ID: "", Type: EventType, Payload: payload}, nil
}

func (v *Verifier) fetchKeys(ctx context.Context) (map[string]*ecdsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.keysURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create github signing keys request: %w", err)
	}
	req.Header.Set("User-Agent", "unkey-secret-scanning")
	res, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch github signing keys: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github signing keys returned HTTP %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, responseBytesMax+1))
	if err != nil {
		return nil, fmt.Errorf("read github signing keys response: %w", err)
	}
	if len(body) > responseBytesMax {
		return nil, errors.New("github signing keys response exceeds size limit")
	}
	return parseKeys(body)
}

func parseKeys(body []byte) (map[string]*ecdsa.PublicKey, error) {
	var response publicKeysResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("decode github signing keys: %w", err)
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
		if !ok || key.Curve != elliptic.P256() {
			continue
		}
		keys[entry.KeyIdentifier] = key
	}
	if len(keys) == 0 {
		return nil, errors.New("github response contains no usable signing keys")
	}
	return keys, nil
}
