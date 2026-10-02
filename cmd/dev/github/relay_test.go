package github

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	githubverifier "github.com/unkeyed/unkey/pkg/webhook/verifiers/github"
)

func TestEventForwarderConfiguration(t *testing.T) {
	for _, test := range []struct {
		name, relay, origin, target, token, admin string
		valid                                     bool
	}{
		{"scoped", "https://relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github", strings.Repeat("s", 43), "", true},
		{"admin", "https://relay.test", "https://dashboard.test", "http://127.0.0.1:7091/webhooks/github", "", "admin", true},
		{"both credentials", "https://relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github", strings.Repeat("s", 43), "admin", false},
		{"no credentials", "https://relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github", "", "", false},
		{"http relay", "http://relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github", "", "admin", false},
		{"origin path", "https://relay.test", "https://dashboard.test/", "http://localhost:7091/webhooks/github", "", "admin", false},
		{"relay credentials", "https://user:secret@relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github", "", "admin", false},
		{"remote receiver", "https://relay.test", "https://dashboard.test", "https://remote.test/webhooks/github", "", "admin", false},
		{"receiver credentials", "https://relay.test", "https://dashboard.test", "http://secret@localhost:7091/webhooks/github", "", "admin", false},
		{"receiver query", "https://relay.test", "https://dashboard.test", "http://localhost:7091/webhooks/github?secret=value", "", "admin", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newEventForwarder(test.relay, test.origin, test.target, test.token, test.admin)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestEventForwarderPreservesSignedDeliveryAndAcknowledgesOnlyAcceptance(t *testing.T) {
	for _, status := range []int{200, 204, 401, 404, 500, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			payload := "{ \"installation\": {\"id\": 123}, \"message\": \"café\\nhello\" }\n"
			delivery := relayDelivery{
				ID: strings.Repeat("d", 43), LeaseToken: strings.Repeat("l", 43),
				Event: "push", DeliveryID: "github-delivery-id", Payload: payload,
				Signature: generateSignature([]byte(payload), "test-webhook-secret"),
			}
			var receiverCalls, acknowledgments, redirectCalls atomic.Int32
			redirectTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				redirectCalls.Add(1)
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(redirectTarget.Close)
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receiverCalls.Add(1)
				event, err := githubverifier.New("test-webhook-secret").Verify(r)
				if err != nil || string(event.Payload) != payload || event.Type != "push" || event.ID != delivery.DeliveryID || r.Header.Get("Authorization") != "" {
					t.Error("signature, raw payload, event, delivery ID, or credential isolation changed")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Location", redirectTarget.URL)
				w.WriteHeader(status)
			}))
			t.Cleanup(receiver.Close)
			relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.Header.Get("Authorization") != "Bearer "+strings.Repeat("s", 43) {
					t.Error("missing scoped credential")
				}
				switch r.URL.Path {
				case "/v1/webhooks/enable":
					w.WriteHeader(http.StatusNoContent)
				case "/v1/webhooks/claim":
					if err := json.NewEncoder(w).Encode(delivery); err != nil {
						t.Error(err)
					}
				case "/v1/webhooks/" + delivery.ID + "/ack":
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["leaseToken"] != delivery.LeaseToken || len(body) != 1 {
						t.Error("incorrect acknowledgment binding")
					}
					acknowledgments.Add(1)
					w.WriteHeader(http.StatusNoContent)
				default:
					t.Errorf("unexpected request %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, receiver.URL+"/webhooks/github")
			delivered, err := f.forwardNext(t.Context())
			if status >= 200 && status < 300 {
				require.NoError(t, err)
				require.True(t, delivered)
				require.EqualValues(t, 1, acknowledgments.Load())
			} else {
				require.Error(t, err)
				require.False(t, delivered)
				require.Zero(t, acknowledgments.Load())
			}
			require.EqualValues(t, 1, receiverCalls.Load())
			require.Zero(t, redirectCalls.Load())
		})
	}
}

func TestEventForwarderEnrollment(t *testing.T) {
	for _, scenario := range []string{"valid", "wrong-origin", "expired", "too-long", "invalid-token"} {
		t.Run(scenario, func(t *testing.T) {
			var enrollments, claims, enables atomic.Int32
			relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/environments" {
					enrollments.Add(1)
					var body struct {
						Origin    string `json:"origin"`
						Reuse     bool   `json:"reuse"`
						ExpiresAt int64  `json:"expiresAt"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || !body.Reuse || body.Origin != "https://dashboard.test" || body.ExpiresAt <= time.Now().UnixMilli() || r.Header.Get("Authorization") != "Bearer admin-secret" {
						t.Error("incorrect reusable enrollment request")
					}
					origin, token, expiry := body.Origin, strings.Repeat("s", 43), time.Now().Add(24*time.Hour).UnixMilli()
					switch scenario {
					case "wrong-origin":
						origin = "https://other.test"
					case "expired":
						expiry = time.Now().Add(-time.Hour).UnixMilli()
					case "too-long":
						expiry = time.Now().Add(31 * 24 * time.Hour).UnixMilli()
					case "invalid-token":
						token = "bad"
					}
					if err := json.NewEncoder(w).Encode(map[string]any{"token": token, "origin": origin, "expiresAt": expiry}); err != nil {
						t.Error(err)
					}
					return
				}
				if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("s", 43) {
					t.Error("admin credential leaked to scoped endpoint")
				}
				if r.URL.Path == "/v1/webhooks/claim" {
					claims.Add(1)
				} else {
					enables.Add(1)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, "http://localhost:7091/webhooks/github")
			f.token, f.adminToken = "", "admin-secret"
			_, err := f.forwardNext(t.Context())
			if scenario != "valid" {
				require.Error(t, err)
				require.Zero(t, claims.Load())
				return
			}
			require.NoError(t, err)
			_, err = f.forwardNext(t.Context())
			require.NoError(t, err)
			require.EqualValues(t, 1, enrollments.Load())
			require.EqualValues(t, 1, enables.Load())
			require.EqualValues(t, 2, claims.Load())
			f.expiresAt = time.Now().Add(time.Hour)
			_, err = f.forwardNext(t.Context())
			require.NoError(t, err)
			require.EqualValues(t, 2, enrollments.Load())
		})
	}
}

func TestEventForwarderBoundsAndRedirects(t *testing.T) {
	var destinationCalls atomic.Int32
	destination := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		destinationCalls.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(destination.Close)
	for _, scenario := range []string{"redirect", "oversize", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if scenario == "redirect" {
					w.Header().Set("Location", destination.URL)
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				if _, err := io.WriteString(w, strings.Repeat("x", relayResponseLimit+1)); err != nil && scenario != "oversize" {
					t.Error(err)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, "http://localhost:7091/webhooks/github")
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if scenario == "cancel" {
				cancel()
			}
			delivered, err := f.forwardNext(ctx)
			require.Error(t, err)
			require.False(t, delivered)
			require.Zero(t, destinationCalls.Load())
		})
	}
}

func TestEventForwarderRejectsInvalidClaimsAndFailedAcknowledgments(t *testing.T) {
	for _, scenario := range []string{"malformed", "unsafe-id", "unknown-event", "ack-failed"} {
		t.Run(scenario, func(t *testing.T) {
			var received, acknowledged atomic.Int32
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				received.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(receiver.Close)
			relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1/webhooks/enable":
					w.WriteHeader(http.StatusNoContent)
				case "/v1/webhooks/claim":
					if scenario == "malformed" {
						if _, err := io.WriteString(w, "not-json-secret"); err != nil {
							t.Error(err)
						}
						return
					}
					delivery := relayDelivery{
						ID: strings.Repeat("d", 43), LeaseToken: strings.Repeat("l", 43),
						DeliveryID: "github-id", Event: "pull_request", Signature: "signature", Payload: "{}",
					}
					if scenario == "unsafe-id" {
						delivery.ID = "../environments"
					}
					if scenario == "unknown-event" {
						delivery.Event = "installation"
					}
					if err := json.NewEncoder(w).Encode(delivery); err != nil {
						t.Error(err)
					}
				default:
					acknowledged.Add(1)
					w.WriteHeader(http.StatusConflict)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, receiver.URL+"/webhooks/github")
			delivered, err := f.forwardNext(t.Context())
			require.Error(t, err)
			require.NotContains(t, err.Error(), "not-json-secret")
			require.False(t, delivered)
			if scenario == "ack-failed" {
				require.EqualValues(t, 1, received.Load())
				require.EqualValues(t, 1, acknowledged.Load())
			} else {
				require.Zero(t, received.Load())
				require.Zero(t, acknowledged.Load())
			}
		})
	}
}

func testEventForwarder(t *testing.T, relay *httptest.Server, target string) *eventForwarder {
	t.Helper()
	f, err := newEventForwarder(relay.URL, "https://dashboard.test", target, strings.Repeat("s", 43), "")
	require.NoError(t, err)
	f.client.Transport = relay.Client().Transport
	t.Cleanup(f.client.CloseIdleConnections)
	return f
}
