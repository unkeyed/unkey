package trufflehog_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/trufflehog"
)

type report struct {
	ID     string `json:"id"`
	Token  string `json:"token"`
	Type   string `json:"type"`
	URL    string `json:"url"`
	Source string `json:"source"`
}

type signer struct {
	key   *ecdsa.PrivateKey
	keyID string
	pem   string
}

func newSigner(t *testing.T, curve elliptic.Curve) signer {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	digest := sha256.Sum256(der)
	return signer{
		key:   key,
		keyID: hex.EncodeToString(digest[:]),
		pem:   string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})),
	}
}

func (s signer) sign(t *testing.T, payload []byte) string {
	t.Helper()
	digest := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, s.key, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

func request(payload []byte, keyID, signature string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	if keyID != "" {
		req.Header.Set("Truffle-Public-Key-Identifier", keyID)
	}
	if signature != "" {
		req.Header.Set("Truffle-Public-Key-Signature", signature)
	}
	return req
}

func reportBody(t *testing.T) []byte {
	t.Helper()
	body, err := json.Marshal([]report{{
		ID: uid.New(uid.TestPrefix), Token: uid.New(uid.TestPrefix), Type: "Unkey",
		URL: "https://github.com/unkeyed/example/blob/abc/.env", Source: "Github",
	}})
	require.NoError(t, err)
	return body
}

func TestPublishedKeyMatchesDocumentedIdentifier(t *testing.T) {
	verifier, err := trufflehog.New(trufflehog.PublicKeyPEM)
	require.NoError(t, err)
	require.Equal(t, "182a7a4194659dcc955a2ab54709183843504dcfd2ba9b7a68e8207b2b1df9a5", verifier.KeyID())
}

func TestVerify(t *testing.T) {
	s := newSigner(t, elliptic.P256())
	verifier, err := trufflehog.New(s.pem)
	require.NoError(t, err)
	require.Equal(t, s.keyID, verifier.KeyID())
	body := reportBody(t)
	signature := s.sign(t, body)

	event, err := verifier.Verify(request(body, s.keyID, signature))
	require.NoError(t, err)
	require.Equal(t, trufflehog.EventType, event.Type)
	require.Equal(t, body, event.Payload)

	tampered := bytes.Clone(body)
	tampered[len(tampered)-2] ^= 1
	_, err = verifier.Verify(request(tampered, s.keyID, signature))
	require.Error(t, err)
	require.NotErrorIs(t, err, webhook.ErrBadRequest)

	other := newSigner(t, elliptic.P256())
	for _, test := range []struct {
		name      string
		keyID     string
		signature string
		malformed bool
	}{
		{name: "missing signature", keyID: s.keyID, malformed: true},
		{name: "missing key identifier", signature: signature, malformed: true},
		{name: "missing both headers", malformed: true},
		{name: "unknown key identifier", keyID: other.keyID, signature: other.sign(t, body)},
		{name: "invalid base64", keyID: s.keyID, signature: "not base64!"},
		{name: "signature from another key", keyID: s.keyID, signature: other.sign(t, body)},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := verifier.Verify(request(body, test.keyID, test.signature))
			require.Error(t, err)
			if test.malformed {
				require.ErrorIs(t, err, webhook.ErrBadRequest)
			} else {
				require.NotErrorIs(t, err, webhook.ErrBadRequest)
			}
		})
	}
}

func TestNewRejectsUnsupportedKeys(t *testing.T) {
	p256DER, err := x509.MarshalPKIXPublicKey(&newSigner(t, elliptic.P256()).key.PublicKey)
	require.NoError(t, err)
	for _, test := range []struct {
		name string
		pem  string
	}{
		{name: "P-384 key", pem: newSigner(t, elliptic.P384()).pem},
		{name: "not PEM", pem: "public key"},
		{name: "wrong PEM block", pem: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: p256DER}))},
		{name: "corrupt key", pem: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("corrupt")}))},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := trufflehog.New(test.pem)
			require.Error(t, err)
		})
	}
}
