package providers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/ssrf"
)

const testDeliveryID = "ndlv_0123456789abcdef0123456789abcdef01234567"

type rewriteTransport struct {
	target *url.URL
}

type receivedRequest struct {
	url    *url.URL
	header http.Header
	body   []byte
}

func (t rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("transport failure with secret data")
}

func testClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	return &http.Client{Transport: rewriteTransport{target: target}, Timeout: requestTimeout}
}

func testRequest() notification.Request {
	return notification.Request{
		DeliveryID: testDeliveryID,
		Event: notification.Event{
			WorkspaceID: "ws", SourceEventID: "source", Type: "test", SchemaVersion: 1,
			Payload: []byte(`{"answer":42}`), OccurredAt: 1,
		},
		Content: notification.Content{Subject: "Subject", Text: "hello <team>", HTML: "<b>hello</b>"},
		Target:  notification.Target{Email: "user@example.com", SigningSecret: "signing-secret"},
	}
}

func sendFailure(t *testing.T, provider notification.Provider, request notification.Request, kind notification.FailureKind, code string, retryAfter time.Duration) {
	t.Helper()
	_, err := provider.Send(t.Context(), request)
	var sendErr *notification.SendError
	require.ErrorAs(t, err, &sendErr)
	require.Equal(t, kind, sendErr.Kind)
	require.Equal(t, code, sendErr.Code)
	require.Equal(t, retryAfter, sendErr.RetryAfter)
	require.NotContains(t, err.Error(), "secret")
}

func TestResendHTTP(t *testing.T) {
	request := testRequest()
	received := make(chan receivedRequest, 1)
	client := testClient(t, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			return
		}
		select {
		case received <- receivedRequest{url: req.URL, header: req.Header.Clone(), body: body}:
		default:
			t.Error("provider sent an extra request")
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"id":"email-id"}`); err != nil {
			t.Error(err)
		}
	})
	provider := &resend{apiKey: "api-key", from: "Sender <sender@example.com>", client: client}
	receipt, err := provider.Send(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, "email-id", receipt.MessageID)
	require.Len(t, received, 1)
	got := <-received
	require.Equal(t, "/emails", got.url.Path)
	require.Equal(t, "Bearer api-key", got.header.Get("Authorization"))
	require.Equal(t, testDeliveryID, got.header.Get("Idempotency-Key"))
	require.JSONEq(t, `{"from":"Sender <sender@example.com>","to":"user@example.com","subject":"Subject","text":"hello <team>","html":"<b>hello</b>"}`, string(got.body))

	request.DeliveryID = "bad\r\nid"
	sendFailure(t, provider, request, notification.Rejected, "invalid_delivery_id", 0)
}

func TestResendConflict(t *testing.T) {
	type testCase struct {
		name string
		body string
		kind notification.FailureKind
		code string
	}
	tests := []testCase{
		{name: "concurrent request", body: `{"statusCode":409,"message":"request in progress","name":"concurrent_idempotent_requests"}`, kind: notification.Ambiguous, code: "concurrent_idempotent_requests"},
		{name: "different payload", body: `{"statusCode":409,"message":"payload differs","name":"invalid_idempotent_request"}`, kind: notification.Rejected, code: "invalid_idempotent_request"},
		{name: "unknown conflict", body: `{"statusCode":409,"message":"secret detail","name":"new_conflict"}`, kind: notification.Rejected, code: "request_rejected"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			received := make(chan string, 1)
			client := testClient(t, func(w http.ResponseWriter, req *http.Request) {
				select {
				case received <- req.Header.Get("Idempotency-Key"):
				default:
					t.Error("provider sent an extra request")
				}
				w.WriteHeader(http.StatusConflict)
				if _, err := io.WriteString(w, test.body); err != nil {
					t.Error(err)
				}
			})
			provider := &resend{apiKey: "api-key", from: "sender@example.com", client: client}
			sendFailure(t, provider, testRequest(), test.kind, test.code, 0)
			require.Len(t, received, 1)
			require.Equal(t, testDeliveryID, <-received)
		})
	}
}

func TestIdempotentPOSTIsNotReplayedAfterResponseEOF(t *testing.T) {
	for _, header := range []string{"Idempotency-Key", "X-Idempotency-Key"} {
		t.Run(header, func(t *testing.T) {
			var posts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodPost {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				posts.Add(1)
				if _, err := io.Copy(io.Discard, req.Body); err != nil {
					t.Error(err)
					return
				}
				connection, _, err := http.NewResponseController(w).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if err := connection.Close(); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)

			response, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())

			req, err := newJSONRequest(t.Context(), server.URL, []byte(`{"event":"accepted"}`))
			require.NoError(t, err)
			req.Header.Set(header, testDeliveryID)
			_, _, _, err = send(server.Client(), req)
			var sendErr *notification.SendError
			require.ErrorAs(t, err, &sendErr)
			require.Equal(t, notification.Ambiguous, sendErr.Kind)
			require.Equal(t, int32(1), posts.Load())
		})
	}
}

func TestWebhookPreSendFailures(t *testing.T) {
	type testCase struct {
		name string
		err  error
		kind notification.FailureKind
	}
	for _, tc := range []testCase{
		{name: "dns", err: &net.DNSError{Err: "unavailable", Name: "private.example", IsTemporary: true}, kind: notification.Retryable},
		{name: "dial", err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}, kind: notification.Retryable},
		{name: "unclassified", err: errors.New("transport failure"), kind: notification.Ambiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.Proxy = nil
			transport.DialContext = func(context.Context, string, string) (net.Conn, error) {
				return nil, tc.err
			}
			t.Cleanup(transport.CloseIdleConnections)
			client := newHTTPClient()
			client.Transport = transport
			request := testRequest()
			request.Target.URL = "https://example.com/hook"
			_, err := (&webhook{client: client}).Send(t.Context(), request)
			var failure *notification.SendError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, tc.kind, failure.Kind)
			require.NotContains(t, err.Error(), "private.example")
		})
	}
	t.Run("forbidden address", func(t *testing.T) {
		request := testRequest()
		request.Target.URL = "https://127.0.0.1/hook"
		_, err := NewWebhook().Send(t.Context(), request)
		var failure *notification.SendError
		require.ErrorAs(t, err, &failure)
		require.Equal(t, notification.Rejected, failure.Kind)
	})
}

func TestWebhookRedirectRemainsAmbiguous(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		http.Redirect(w, req, "https://127.0.0.1/redirect", http.StatusFound)
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	client := ssrf.New(ssrf.WithTimeout(requestTimeout))
	client.Transport = rewriteTransport{target: target}
	request := testRequest()
	request.Target.URL = "https://example.com/hook"
	_, err = (&webhook{client: client}).Send(t.Context(), request)
	var failure *notification.SendError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, notification.Ambiguous, failure.Kind)
	require.Equal(t, int32(1), requests.Load())
}

func TestWebhookDialTimeoutIsRetryable(t *testing.T) {
	release := make(chan struct{})
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
		select {
		case <-release:
			return nil, errors.New("dial released")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	t.Cleanup(func() {
		close(release)
		transport.CloseIdleConnections()
	})
	client := newHTTPClient()
	client.Transport = transport
	client.Timeout = 100 * time.Millisecond
	request := testRequest()
	request.Target.URL = "https://example.com/hook"
	_, err := (&webhook{client: client}).Send(t.Context(), request)
	var failure *notification.SendError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, notification.Retryable, failure.Kind)
}

func TestWebhookResponseTimeoutRemainsAmbiguous(t *testing.T) {
	release := make(chan struct{})
	var requests atomic.Int32
	client := testClient(t, func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
		<-release
	})
	t.Cleanup(func() { close(release) })
	client.Timeout = 100 * time.Millisecond
	request := testRequest()
	request.Target.URL = "https://example.com/hook"
	_, err := (&webhook{client: client}).Send(t.Context(), request)
	var failure *notification.SendError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, notification.Ambiguous, failure.Kind)
	require.Equal(t, int32(1), requests.Load())
}

func TestSlackHTTP(t *testing.T) {
	request := testRequest()
	request.Target.URL = "https://hooks.slack.com/services/T/B/token"
	received := make(chan receivedRequest, 1)
	client := testClient(t, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			return
		}
		select {
		case received <- receivedRequest{url: req.URL, body: body}:
		default:
			t.Error("provider sent an extra request")
		}
		if _, err := io.WriteString(w, "ok"); err != nil {
			t.Error(err)
		}
	})
	provider := &slack{client: client}
	_, err := provider.Send(t.Context(), request)
	require.NoError(t, err)
	require.Len(t, received, 1)
	got := <-received
	require.Equal(t, "/services/T/B/token", got.url.Path)
	require.JSONEq(t, `{"text":"Subject\nhello &lt;team&gt;"}`, string(got.body))

	request.Target.URL = "https://hooks.slack.com:443/services/T/B/token"
	sendFailure(t, provider, request, notification.Rejected, "invalid_webhook_url", 0)
	request.Target.URL = "https://hooks.slack.com/services/T/B/token"
	request.Content.Text = strings.Repeat("x", slackContentMax+1)
	request.Content.Subject = ""
	sendFailure(t, provider, request, notification.Rejected, "content_too_large", 0)
}

func TestDiscordHTTP(t *testing.T) {
	request := testRequest()
	request.Target.URL = "https://discord.com/api/webhooks/123/token"
	received := make(chan receivedRequest, 1)
	client := testClient(t, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			return
		}
		select {
		case received <- receivedRequest{url: req.URL, body: body}:
		default:
			t.Error("provider sent an extra request")
		}
		if _, err := io.WriteString(w, `{"id":"message-id"}`); err != nil {
			t.Error(err)
		}
	})
	provider := &discord{client: client}
	receipt, err := provider.Send(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, "message-id", receipt.MessageID)
	require.Len(t, received, 1)
	got := <-received
	require.Equal(t, "wait=true", got.url.RawQuery)
	require.JSONEq(t, `{"content":"Subject\nhello <team>","allowed_mentions":{"parse":[]}}`, string(got.body))

	request.Target.URL = "https://discord.com:443/api/webhooks/123/token"
	sendFailure(t, provider, request, notification.Rejected, "invalid_webhook_url", 0)
}

func TestWebhookHTTPAndSignature(t *testing.T) {
	request := testRequest()
	request.Target.URL = "https://example.com/notify"
	request.Target.Headers = map[string]string{"X-Tenant": "tenant"}
	received := make(chan receivedRequest, 1)
	client := testClient(t, func(w http.ResponseWriter, req *http.Request) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Error(err)
			return
		}
		select {
		case received <- receivedRequest{header: req.Header.Clone(), body: body}:
		default:
			t.Error("provider sent an extra request")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	provider := &webhook{client: client}
	_, err := provider.Send(t.Context(), request)
	require.NoError(t, err)
	require.Len(t, received, 1)
	got := <-received
	require.Equal(t, testDeliveryID, got.header.Get("Unkey-Delivery-Id"))
	require.Equal(t, "tenant", got.header.Get("X-Tenant"))
	timestamp := got.header.Get("Unkey-Timestamp")
	_, err = time.Parse(time.RFC3339, timestamp)
	require.NoError(t, err)
	mac := hmac.New(sha256.New, []byte("signing-secret"))
	_, err = mac.Write(append([]byte(timestamp+"."), got.body...))
	require.NoError(t, err)
	require.Equal(t, signaturePrefix+hex.EncodeToString(mac.Sum(nil)), got.header.Get("Unkey-Signature"))
	require.JSONEq(t, `{"deliveryId":"ndlv_0123456789abcdef0123456789abcdef01234567","event":{"workspaceId":"ws","sourceEventId":"source","type":"test","schemaVersion":1,"resource":{},"payload":{"answer":42},"occurredAt":1},"content":{"subject":"Subject","text":"hello <team>","html":"<b>hello</b>"}}`, string(got.body))

	request.Target.Headers = map[string]string{"Unkey-Signature": "replacement"}
	sendFailure(t, provider, request, notification.Rejected, "reserved_webhook_header", 0)
	request.Target.Headers = map[string]string{"X-Test": "bad\x00value"}
	sendFailure(t, provider, request, notification.Rejected, "invalid_webhook_header", 0)
	request.Target.URL = "http://127.0.0.1/notify"
	sendFailure(t, provider, request, notification.Rejected, "invalid_webhook_url", 0)
}

func TestProviderHTTPFailures(t *testing.T) {
	type providerCase struct {
		provider          func(*http.Client) notification.Provider
		request           func() notification.Request
		validatesResponse bool
	}
	tests := map[string]providerCase{
		"resend": {provider: func(client *http.Client) notification.Provider {
			return &resend{apiKey: "key", from: "sender@example.com", client: client}
		}, request: testRequest, validatesResponse: true},
		"slack": {provider: func(client *http.Client) notification.Provider { return &slack{client: client} }, request: func() notification.Request {
			request := testRequest()
			request.Target.URL = "https://hooks.slack.com/services/T/B/token"
			return request
		}, validatesResponse: true},
		"discord": {provider: func(client *http.Client) notification.Provider { return &discord{client: client} }, request: func() notification.Request {
			request := testRequest()
			request.Target.URL = "https://discord.com/api/webhooks/123/token"
			return request
		}, validatesResponse: true},
		"webhook": {provider: func(client *http.Client) notification.Provider { return &webhook{client: client} }, request: func() notification.Request {
			request := testRequest()
			request.Target.URL = "https://example.com/notify"
			return request
		}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			request := test.request()
			provider := test.provider(&http.Client{Transport: failingTransport{}})
			sendFailure(t, provider, request, notification.Ambiguous, "request_failed", 0)

			type responseCase struct {
				name       string
				status     int
				body       string
				retryAfter string
				kind       notification.FailureKind
				code       string
				delay      time.Duration
			}
			cases := []responseCase{
				{name: "malformed success", status: http.StatusOK, body: "malformed", kind: notification.Ambiguous, code: "invalid_response"},
				{name: "rejected", status: http.StatusBadRequest, kind: notification.Rejected, code: "request_rejected"},
				{name: "request timeout", status: http.StatusRequestTimeout, kind: notification.Ambiguous, code: "request_timeout"},
				{name: "server error", status: http.StatusServiceUnavailable, kind: notification.Ambiguous, code: "provider_unavailable"},
				{name: "rate limit default", status: http.StatusTooManyRequests, retryAfter: "invalid", kind: notification.Retryable, code: "rate_limited", delay: retryAfterDefault},
				{name: "rate limit extreme", status: http.StatusTooManyRequests, retryAfter: "9223372036854775807", kind: notification.Retryable, code: "rate_limited", delay: retryAfterMax},
			}
			for _, testCase := range cases {
				if testCase.name == "malformed success" && !test.validatesResponse {
					continue
				}
				t.Run(testCase.name, func(t *testing.T) {
					client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
						w.Header().Set("Retry-After", testCase.retryAfter)
						w.WriteHeader(testCase.status)
						if _, err := io.WriteString(w, testCase.body); err != nil {
							t.Error(err)
						}
					})
					sendFailure(t, test.provider(client), request, testCase.kind, testCase.code, testCase.delay)
				})
			}
		})
	}
}
