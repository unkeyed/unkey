package trufflehog

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/unkeyed/unkey/pkg/webhook"
)

// EventType identifies partner revocation reports, which carry no event-type header.
const EventType = "trufflehog_secret_scanning"

// PublicKeyPEM is the key TruffleHog signs partner revocation requests with.
// TruffleHog publishes it in its documentation and does not rotate it.
const PublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEhsTGPMKvLlzNL5mK1G63KabIWkV3
zNtcMdQZNbrgv/9Tl1A5ZSfoD3pyEEUpK6EnDj5xbmrZC7Hju+fKcte/0g==
-----END PUBLIC KEY-----`

// Verifier is safe for concurrent use and must be constructed with New.
type Verifier struct {
	keyID string
	key   *ecdsa.PublicKey
}

var _ webhook.Verifier = (*Verifier)(nil)

// New parses a PEM-encoded ECDSA P-256 public key. The key identifier that
// requests must present is the hex SHA-256 of its DER SubjectPublicKeyInfo.
func New(publicKeyPEM string) (*Verifier, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("trufflehog public key is not a PEM public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse trufflehog public key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, errors.New("trufflehog public key is not ECDSA P-256")
	}
	digest := sha256.Sum256(block.Bytes)
	return &Verifier{keyID: hex.EncodeToString(digest[:]), key: key}, nil
}

// KeyID returns the identifier requests must present for this verifier's key.
func (v *Verifier) KeyID() string {
	return v.keyID
}

// Verify authenticates the raw body against the ASN.1 ECDSA signature in
// Truffle-Public-Key-Signature.
func (v *Verifier) Verify(r *http.Request) (webhook.Event, error) {
	signature := r.Header.Get("Truffle-Public-Key-Signature")
	keyID := r.Header.Get("Truffle-Public-Key-Identifier")
	if signature == "" || keyID == "" {
		return webhook.Event{}, fmt.Errorf("%w: missing signing headers", webhook.ErrBadRequest)
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return webhook.Event{}, err
	}
	if keyID != v.keyID {
		return webhook.Event{}, errors.New("unknown trufflehog signing key")
	}
	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil || len(signatureBytes) == 0 {
		return webhook.Event{}, errors.New("invalid trufflehog signature encoding")
	}

	digest := sha256.Sum256(payload)
	if !ecdsa.VerifyASN1(v.key, digest[:], signatureBytes) {
		return webhook.Event{}, errors.New("trufflehog signature mismatch")
	}
	return webhook.Event{ID: "", Type: EventType, Payload: payload}, nil
}
