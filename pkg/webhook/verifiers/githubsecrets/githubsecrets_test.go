package githubsecrets_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/webhook"
	"github.com/unkeyed/unkey/pkg/webhook/verifiers/githubsecrets"
)

type publicKey struct {
	ID      string `json:"key_identifier"`
	Key     string `json:"key"`
	Current bool   `json:"is_current"`
}

type keysResponse struct {
	Keys []publicKey `json:"public_keys"`
}

func verify(verifier *githubsecrets.Verifier, ctx context.Context, payload []byte, signature, keyID string) (webhook.Event, error) {
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/", bytes.NewReader(payload))
	req.Header.Set("Github-Public-Key-Signature", signature)
	req.Header.Set("Github-Public-Key-Identifier", keyID)
	return verifier.Verify(req)
}

func signingKey(t *testing.T, curve elliptic.Curve) (*ecdsa.PrivateKey, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	return key, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func sign(t *testing.T, key *ecdsa.PrivateKey, payload []byte) string {
	t.Helper()
	digest := sha256.Sum256(payload)
	signature, err := ecdsa.SignASN1(rand.Reader, key, digest[:])
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(signature)
}

func serveKeys(t *testing.T, response keysResponse, count *atomic.Int32) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(response)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if r.Header.Get("User-Agent") != "unkey-secret-scanning" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, writeErr := w.Write(body)
		if writeErr != nil {
			t.Error(writeErr)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestVerifier_VerifiesExactBytesAndRejectsInvalidSignatures(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	forger, _ := signingKey(t, elliptic.P256())
	payload := []byte(" exact\x00payload\r\n")
	signature := sign(t, key, payload)
	var count atomic.Int32
	server := serveKeys(t, keysResponse{Keys: []publicKey{
		{ID: "active", Key: encoded},
		{ID: "malformed", Key: "not PEM"},
	}}, &count)
	verifier := githubsecrets.New(server.URL, clock.NewTestClock())
	for _, test := range []struct {
		name      string
		payload   []byte
		signature string
		keyID     string
		valid     bool
	}{
		{name: "valid", payload: payload, signature: signature, keyID: "active", valid: true},
		{name: "tampered", payload: []byte("exact\x00payload\n"), signature: signature, keyID: "active"},
		{name: "forged", payload: payload, signature: sign(t, forger, payload), keyID: "active"},
		{name: "malformed base64", payload: payload, signature: "!", keyID: "active"},
		{name: "malformed ASN1", payload: payload, signature: base64.StdEncoding.EncodeToString([]byte("invalid")), keyID: "active"},
		{name: "malformed public key", payload: payload, signature: signature, keyID: "malformed"},
		{name: "unknown key", payload: payload, signature: signature, keyID: "missing"},
		{name: "empty key", payload: payload, signature: signature},
	} {
		t.Run(test.name, func(t *testing.T) {
			event, err := verify(verifier, t.Context(), test.payload, test.signature, test.keyID)
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, githubsecrets.EventType, event.Type)
				require.Equal(t, test.payload, event.Payload)
			} else {
				require.Error(t, err)
				require.NotErrorIs(t, err, webhook.ErrUnavailable)
				require.Empty(t, event.Payload)
			}
		})
	}
	require.EqualValues(t, 1, count.Load())
}

func TestVerifier_RotatesAfterExpiryAndPreservesStaleKeysOnFailure(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	rotated, encodedRotated := signingKey(t, elliptic.P256())
	payload := []byte("payload")
	signature := sign(t, key, payload)
	rotatedSignature := sign(t, rotated, payload)
	var count atomic.Int32
	var phase atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		if phase.Load() == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		entry := publicKey{ID: "initial", Key: encoded}
		if phase.Load() == 2 {
			entry = publicKey{ID: "rotated", Key: encodedRotated}
		}
		if err := json.NewEncoder(w).Encode(keysResponse{Keys: []publicKey{entry}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := githubsecrets.New(server.URL, testClock)
	_, err := verify(verifier, t.Context(), payload, signature, "initial")
	require.NoError(t, err)
	phase.Store(1)
	testClock.Tick(time.Hour)
	for range 10 {
		_, err = verify(verifier, t.Context(), payload, signature, "initial")
		require.NoError(t, err)
	}
	require.EqualValues(t, 2, count.Load())
	phase.Store(2)
	_, err = verify(verifier, t.Context(), payload, rotatedSignature, "rotated")
	require.Error(t, err)
	testClock.Tick(time.Minute)
	_, err = verify(verifier, t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	_, err = verify(verifier, t.Context(), payload, signature, "initial")
	require.Error(t, err)
	require.EqualValues(t, 3, count.Load())
	testClock.Tick(time.Hour - time.Second)
	_, err = verify(verifier, t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.EqualValues(t, 3, count.Load())
	testClock.Tick(time.Second)
	_, err = verify(verifier, t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.EqualValues(t, 4, count.Load())
}

func TestVerifier_UnavailableKeysAreRetriedAtMostOncePerMinute(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := githubsecrets.New(server.URL, testClock)
	for range 10 {
		_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
		require.ErrorIs(t, err, webhook.ErrUnavailable)
	}
	require.EqualValues(t, 1, count.Load())
	testClock.Tick(time.Minute)
	_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
	require.EqualValues(t, 2, count.Load())
}

func TestVerifier_RejectsMalformedAndWrongCurveKeys(t *testing.T) {
	key, _ := signingKey(t, elliptic.P256())
	_, wrongCurve := signingKey(t, elliptic.P384())
	for _, encoded := range []string{"not PEM", wrongCurve} {
		var count atomic.Int32
		server := serveKeys(t, keysResponse{Keys: []publicKey{{ID: "invalid", Key: encoded}}}, &count)
		verifier := githubsecrets.New(server.URL, clock.NewTestClock())
		_, err := verify(verifier, t.Context(), nil, sign(t, key, nil), "invalid")
		require.ErrorIs(t, err, webhook.ErrUnavailable)
	}
}

func TestVerifier_BoundsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(strings.Repeat(" ", (1<<20)+1))); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	verifier := githubsecrets.New(server.URL, clock.NewTestClock())
	_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
}

func TestVerifier_UnreachableEndpointReturnsErrorWithoutCache(t *testing.T) {
	var count atomic.Int32
	server := serveKeys(t, keysResponse{}, &count)
	server.Close()
	verifier := githubsecrets.New(server.URL, clock.NewTestClock())
	_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
}

func TestVerifier_CoalescesFetchesAndSupportsConcurrentCachedUse(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	var count atomic.Int32
	server := serveKeys(t, keysResponse{Keys: []publicKey{{ID: "active", Key: encoded}}}, &count)
	verifier := githubsecrets.New(server.URL, clock.NewTestClock())
	signature := sign(t, key, nil)
	for range 2 {
		var group sync.WaitGroup
		start := make(chan struct{})
		results := make(chan bool, 100)
		for range 100 {
			group.Go(func() {
				<-start
				_, err := verify(verifier, context.Background(), nil, signature, "active")
				results <- err == nil
			})
		}
		close(start)
		group.Wait()
		close(results)
		for valid := range results {
			require.True(t, valid)
		}
		require.EqualValues(t, 1, count.Load())
	}
}
