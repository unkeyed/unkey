package githubsignature_test

import (
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
	"github.com/unkeyed/unkey/svc/ctrl/internal/githubsignature"
)

type publicKey struct {
	ID      string `json:"key_identifier"`
	Key     string `json:"key"`
	Current bool   `json:"is_current"`
}

type keysResponse struct {
	Keys []publicKey `json:"public_keys"`
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
	verifier := githubsignature.New(server.URL, clock.NewTestClock())
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
			valid, err := verifier.Verify(t.Context(), test.payload, test.signature, test.keyID)
			require.NoError(t, err)
			require.Equal(t, test.valid, valid)
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
	verifier := githubsignature.New(server.URL, testClock)
	valid, err := verifier.Verify(t.Context(), payload, signature, "initial")
	require.NoError(t, err)
	require.True(t, valid)
	phase.Store(1)
	testClock.Tick(time.Hour)
	for range 10 {
		valid, err = verifier.Verify(t.Context(), payload, signature, "initial")
		require.NoError(t, err)
		require.True(t, valid)
	}
	require.EqualValues(t, 2, count.Load())
	phase.Store(2)
	valid, err = verifier.Verify(t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.False(t, valid)
	testClock.Tick(time.Minute)
	valid, err = verifier.Verify(t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.True(t, valid)
	valid, err = verifier.Verify(t.Context(), payload, signature, "initial")
	require.NoError(t, err)
	require.False(t, valid)
	require.EqualValues(t, 3, count.Load())
	testClock.Tick(time.Hour - time.Second)
	valid, err = verifier.Verify(t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.True(t, valid)
	require.EqualValues(t, 3, count.Load())
	testClock.Tick(time.Second)
	valid, err = verifier.Verify(t.Context(), payload, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.True(t, valid)
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
	verifier := githubsignature.New(server.URL, testClock)
	for range 10 {
		valid, err := verifier.Verify(t.Context(), nil, "AA==", "unknown")
		require.Error(t, err)
		require.False(t, valid)
	}
	require.EqualValues(t, 1, count.Load())
	testClock.Tick(time.Minute)
	valid, err := verifier.Verify(t.Context(), nil, "AA==", "unknown")
	require.Error(t, err)
	require.False(t, valid)
	require.EqualValues(t, 2, count.Load())
}

func TestVerifier_RejectsMalformedAndWrongCurveKeys(t *testing.T) {
	key, _ := signingKey(t, elliptic.P256())
	_, wrongCurve := signingKey(t, elliptic.P384())
	for _, encoded := range []string{"not PEM", wrongCurve} {
		var count atomic.Int32
		server := serveKeys(t, keysResponse{Keys: []publicKey{{ID: "invalid", Key: encoded}}}, &count)
		verifier := githubsignature.New(server.URL, clock.NewTestClock())
		valid, err := verifier.Verify(t.Context(), nil, sign(t, key, nil), "invalid")
		require.Error(t, err)
		require.False(t, valid)
	}
}

func TestVerifier_BoundsResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(strings.Repeat(" ", (1<<20)+1))); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	verifier := githubsignature.New(server.URL, clock.NewTestClock())
	valid, err := verifier.Verify(t.Context(), nil, "AA==", "unknown")
	require.Error(t, err)
	require.False(t, valid)
}

func TestVerifier_UnreachableEndpointReturnsErrorWithoutCache(t *testing.T) {
	var count atomic.Int32
	server := serveKeys(t, keysResponse{}, &count)
	server.Close()
	verifier := githubsignature.New(server.URL, clock.NewTestClock())
	valid, err := verifier.Verify(t.Context(), nil, "AA==", "unknown")
	require.Error(t, err)
	require.False(t, valid)
}

func TestVerifier_CoalescesFetchesAndSupportsConcurrentCachedUse(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	var count atomic.Int32
	server := serveKeys(t, keysResponse{Keys: []publicKey{{ID: "active", Key: encoded}}}, &count)
	verifier := githubsignature.New(server.URL, clock.NewTestClock())
	signature := sign(t, key, nil)
	for range 2 {
		var group sync.WaitGroup
		start := make(chan struct{})
		results := make(chan bool, 100)
		for range 100 {
			group.Go(func() {
				<-start
				valid, err := verifier.Verify(context.Background(), nil, signature, "active")
				results <- valid && err == nil
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
