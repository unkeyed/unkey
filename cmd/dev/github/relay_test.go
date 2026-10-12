package github

import (
	"context"
	"encoding/json"
	"fmt"
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
				if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("s", 43) {
					t.Error("missing scoped credential")
				}
				switch r.URL.Path {
				case "/v1/webhooks/enable":
					w.WriteHeader(http.StatusNoContent)
				case "/v1/webhooks/stream":
					if r.Method != http.MethodGet || r.Header.Get("Accept") != "text/event-stream" {
						t.Error("invalid stream request")
					}
					writeRelayDelivery(t, w, delivery)
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
			err := f.forwardStream(t.Context())
			if status >= 200 && status < 300 {
				require.ErrorIs(t, err, io.EOF)
				require.EqualValues(t, 1, acknowledgments.Load())
			} else {
				require.Error(t, err)
				require.NotErrorIs(t, err, io.EOF)
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
			var enrollments, streams, enables atomic.Int32
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
				if r.URL.Path == "/v1/webhooks/stream" {
					streams.Add(1)
					w.Header().Set("Content-Type", "text/event-stream")
					w.WriteHeader(http.StatusOK)
				} else {
					enables.Add(1)
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, "http://localhost:7091/webhooks/github")
			f.token, f.adminToken = "", "admin-secret"
			err := f.forwardStream(t.Context())
			if scenario != "valid" {
				require.Error(t, err)
				require.Zero(t, streams.Load())
				return
			}
			require.ErrorIs(t, err, io.EOF)
			err = f.forwardStream(t.Context())
			require.ErrorIs(t, err, io.EOF)
			require.EqualValues(t, 1, enrollments.Load())
			require.EqualValues(t, 1, enables.Load())
			require.EqualValues(t, 2, streams.Load())
			f.expiresAt = time.Now().Add(time.Hour)
			err = f.forwardStream(t.Context())
			require.ErrorIs(t, err, io.EOF)
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
				w.Header().Set("Content-Type", "text/event-stream")
				if _, err := io.WriteString(w, strings.Repeat("x", relayResponseLimit+1)); err != nil && scenario != "oversize" {
					t.Error(err)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, "http://localhost:7091/webhooks/github")
			f.enabled = true
			ctx, cancel := context.WithCancel(t.Context())
			t.Cleanup(cancel)
			if scenario == "cancel" {
				cancel()
			}
			err := f.forwardStream(ctx)
			require.Error(t, err)
			require.NotErrorIs(t, err, io.EOF)
			require.Zero(t, destinationCalls.Load())
		})
	}
}

func TestEventForwarderRejectsInvalidEventsAndFailedAcknowledgments(t *testing.T) {
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
				case "/v1/webhooks/stream":
					w.Header().Set("Content-Type", "text/event-stream")
					if scenario == "malformed" {
						if _, err := io.WriteString(w, "event: delivery\ndata: not-json-secret\n\n"); err != nil {
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
					writeRelayDelivery(t, w, delivery)
				default:
					acknowledged.Add(1)
					w.WriteHeader(http.StatusConflict)
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, receiver.URL+"/webhooks/github")
			err := f.forwardStream(t.Context())
			require.Error(t, err)
			require.NotErrorIs(t, err, io.EOF)
			require.NotContains(t, err.Error(), "not-json-secret")
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

func TestEventForwarderSSEFraming(t *testing.T) {
	delivery := relayDelivery{
		ID: strings.Repeat("d", 43), LeaseToken: strings.Repeat("l", 43),
		DeliveryID: "github-id", Event: "push", Signature: "signature", Payload: "{\"text\":\"café\\nhello\"}",
	}
	body, err := json.Marshal(delivery)
	require.NoError(t, err)
	split := strings.IndexByte(string(body), ',') + 1
	for _, test := range []struct {
		name, frame string
		deliveries  int32
	}{
		{"multiline CRLF and heartbeat", ": heartbeat\r\n\r\nevent: delivery\r\ndata: " + string(body[:split]) + "\r\ndata: " + string(body[split:]) + "\r\n\r\n", 1},
		{"truncated event", "event: delivery\ndata: " + string(body) + "\n", 0},
		{"unrecognized event", "event: heartbeat\ndata: {}\n\n", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			var received, acknowledged atomic.Int32
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				payload, err := io.ReadAll(r.Body)
				if err != nil || string(payload) != delivery.Payload {
					t.Error("payload changed during SSE decoding")
				}
				received.Add(1)
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(receiver.Close)
			relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/webhooks/stream" {
					acknowledged.Add(1)
					w.WriteHeader(http.StatusNoContent)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
				for _, part := range strings.SplitAfter(test.frame, ":") {
					if _, err := io.WriteString(w, part); err != nil {
						t.Error(err)
						return
					}
					if err := http.NewResponseController(w).Flush(); err != nil {
						t.Error(err)
						return
					}
				}
			}))
			t.Cleanup(relay.Close)
			f := testEventForwarder(t, relay, receiver.URL+"/webhooks/github")
			f.enabled = true
			require.ErrorIs(t, f.forwardStream(t.Context()), io.EOF)
			require.Equal(t, test.deliveries, received.Load())
			require.Equal(t, test.deliveries, acknowledged.Load())
		})
	}
}

func TestEventForwarderKeepsOneStreamAcrossDeliveriesAndCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	var received, streams atomic.Int32
	acked := make(chan struct{}, 2)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(receiver.Close)
	relay := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/webhooks/stream" {
			acked <- struct{}{}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		streams.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		if _, err := io.WriteString(w, ": heartbeat\n\n"); err != nil {
			t.Error(err)
			return
		}
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Error(err)
			return
		}
		select {
		case <-time.After(500 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
		for _, id := range []string{strings.Repeat("a", 43), strings.Repeat("b", 43)} {
			writeRelayDelivery(t, w, relayDelivery{
				ID: id, LeaseToken: strings.Repeat("l", 43), Event: "push",
				DeliveryID: id, Signature: "signature", Payload: "{}",
			})
			if err := http.NewResponseController(w).Flush(); err != nil {
				t.Error(err)
				return
			}
			select {
			case <-acked:
			case <-r.Context().Done():
				return
			}
		}
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(relay.Close)
	f := testEventForwarder(t, relay, receiver.URL+"/webhooks/github")
	f.enabled = true
	f.client.Timeout = 200 * time.Millisecond
	require.Error(t, f.forwardStream(ctx))
	require.EqualValues(t, 2, received.Load())
	require.EqualValues(t, 1, streams.Load())
}

func testEventForwarder(t *testing.T, relay *httptest.Server, target string) *eventForwarder {
	t.Helper()
	f, err := newEventForwarder(relay.URL, "https://dashboard.test", target, strings.Repeat("s", 43), "")
	require.NoError(t, err)
	f.client.Transport = relay.Client().Transport
	t.Cleanup(f.client.CloseIdleConnections)
	return f
}

func writeRelayDelivery(t *testing.T, w http.ResponseWriter, delivery relayDelivery) {
	t.Helper()
	body, err := json.Marshal(delivery)
	if err != nil {
		t.Error(err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	if _, err := fmt.Fprintf(w, "event: delivery\ndata: %s\n\n", body); err != nil {
		t.Error(err)
	}
}
