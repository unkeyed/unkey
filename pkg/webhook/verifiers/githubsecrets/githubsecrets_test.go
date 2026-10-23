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

func newVerifier(t *testing.T, keysURL string, clk clock.Clock) *githubsecrets.Verifier {
	t.Helper()
	verifier, err := githubsecrets.New(keysURL, clk)
	require.NoError(t, err)
	t.Cleanup(verifier.Close)
	return verifier
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
	verifier := newVerifier(t, server.URL, clock.NewTestClock())
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
	verifier := newVerifier(t, server.URL, testClock)
	_, err := verify(verifier, t.Context(), payload, signature, "initial")
	require.NoError(t, err)
	testClock.Tick(time.Hour - time.Second)
	_, err = verify(verifier, t.Context(), payload, signature, "initial")
	require.NoError(t, err)
	require.EqualValues(t, 1, count.Load())

	phase.Store(1)
	testClock.Tick(time.Second)
	_, err = verify(verifier, t.Context(), payload, rotatedSignature, "rotated")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
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
}

func TestVerifier_UnavailableKeysAreRetriedAtMostOncePerMinute(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := newVerifier(t, server.URL, testClock)
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
		verifier := newVerifier(t, server.URL, clock.NewTestClock())
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
	verifier := newVerifier(t, server.URL, clock.NewTestClock())
	_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
}

func TestVerifier_UnreachableEndpointReturnsErrorWithoutCache(t *testing.T) {
	var count atomic.Int32
	server := serveKeys(t, keysResponse{}, &count)
	server.Close()
	verifier := newVerifier(t, server.URL, clock.NewTestClock())
	_, err := verify(verifier, t.Context(), nil, "AA==", "unknown")
	require.ErrorIs(t, err, webhook.ErrUnavailable)
}

func TestVerifier_CoalescesFetchesAndSupportsConcurrentCachedUse(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	var count atomic.Int32
	server := serveKeys(t, keysResponse{Keys: []publicKey{{ID: "active", Key: encoded}}}, &count)
	verifier := newVerifier(t, server.URL, clock.NewTestClock())
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

func TestVerifier_StaleKeysDoNotWaitForRefresh(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	rotated, encodedRotated := signingKey(t, elliptic.P256())
	var count atomic.Int32
	refreshing := make(chan struct{})
	release := make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := publicKey{ID: "initial", Key: encoded}
		if count.Add(1) > 1 {
			close(refreshing)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
			entry = publicKey{ID: "rotated", Key: encodedRotated}
		}
		if err := json.NewEncoder(w).Encode(keysResponse{Keys: []publicKey{entry}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := newVerifier(t, server.URL, testClock)
	t.Cleanup(unblock)
	signature := sign(t, key, nil)
	_, err := verify(verifier, t.Context(), nil, signature, "initial")
	require.NoError(t, err)

	testClock.Tick(time.Hour)
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	_, err = verify(verifier, ctx, nil, signature, "initial")
	require.NoError(t, err)
	select {
	case <-refreshing:
	case <-ctx.Done():
		t.Fatal("stale keys did not trigger refresh")
	}
	_, err = verify(verifier, ctx, nil, signature, "initial")
	require.NoError(t, err, "cached verification must not wait for the refresh")

	waitCtx, stopWaiting := context.WithTimeout(t.Context(), 50*time.Millisecond)
	t.Cleanup(stopWaiting)
	rotatedSignature := sign(t, rotated, nil)
	_, err = verify(verifier, waitCtx, nil, rotatedSignature, "rotated")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	unblock()
	_, err = verify(verifier, t.Context(), nil, rotatedSignature, "rotated")
	require.NoError(t, err)
	require.EqualValues(t, 2, count.Load())
}

func TestVerifier_DiscardsStaleKeysAfter24Hours(t *testing.T) {
	key, encoded := signingKey(t, elliptic.P256())
	var unavailable atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if unavailable.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if err := json.NewEncoder(w).Encode(keysResponse{Keys: []publicKey{{ID: "active", Key: encoded}}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := newVerifier(t, server.URL, testClock)
	signature := sign(t, key, nil)
	_, err := verify(verifier, t.Context(), nil, signature, "active")
	require.NoError(t, err)

	unavailable.Store(true)
	testClock.Tick(24*time.Hour - time.Second)
	_, err = verify(verifier, t.Context(), nil, signature, "active")
	require.NoError(t, err)
	testClock.Tick(time.Second)
	_, err = verify(verifier, t.Context(), nil, signature, "active")
	require.ErrorIs(t, err, webhook.ErrUnavailable)

	unavailable.Store(false)
	testClock.Tick(time.Minute)
	_, err = verify(verifier, t.Context(), nil, signature, "active")
	require.NoError(t, err)
}

func TestVerifier_UnknownKeyRefreshIsRateLimited(t *testing.T) {
	initial, encodedInitial := signingKey(t, elliptic.P256())
	rotated, encodedRotated := signingKey(t, elliptic.P256())
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entry := publicKey{ID: "initial", Key: encodedInitial}
		if count.Add(1) > 1 {
			entry = publicKey{ID: "rotated", Key: encodedRotated}
		}
		if err := json.NewEncoder(w).Encode(keysResponse{Keys: []publicKey{entry}}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	testClock := clock.NewTestClock()
	verifier := newVerifier(t, server.URL, testClock)
	_, err := verify(verifier, t.Context(), nil, sign(t, initial, nil), "initial")
	require.NoError(t, err)

	signature := sign(t, rotated, nil)
	for range 10 {
		_, err = verify(verifier, t.Context(), nil, signature, "rotated")
		require.Error(t, err)
		require.NotErrorIs(t, err, webhook.ErrUnavailable)
	}
	require.EqualValues(t, 1, count.Load())

	testClock.Tick(time.Minute)
	for range 10 {
		_, err = verify(verifier, t.Context(), nil, signature, "rotated")
		require.NoError(t, err)
	}
	require.EqualValues(t, 2, count.Load())
}
