package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/paseto"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
)

const (
	metadataSigningKey          = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"
	alternateMetadataSigningKey = "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"
)

func TestRequestHops_WithoutMetadataStartsWithEmptyHistory(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	hops, err := applyPeerMetadata(newMetadataSession(t, req), nil, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
}

func TestRequestHops_VerifiesAndRemovesMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	req := requestWithMetadata(t, codec, metadata)

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Equal(t, metadata.Hops, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_RestoresOnlyVerifiedIngressRequestID(t *testing.T) {
	for _, test := range []struct {
		name      string
		trustedID bool
	}{
		{"valid", true},
		{"missing", false},
		{"malformed", false},
		{"wrong key", false},
		{"duplicate", false},
		{"expired", false},
		{"missing expiry", false},
		{"invalid client IP", false},
		{"no hops", false},
		{"empty ingress ID", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			codec := newMetadataCodec(t, metadataSigningKey)
			metadata := validMetadata()
			ingressID := uid.New(uid.RequestPrefix)
			metadata.Hops[0].RequestID = ingressID
			metadata.Hops = append(metadata.Hops, meta.Hop{RequestID: uid.New(uid.RequestPrefix)})
			signer := codec
			switch test.name {
			case "wrong key":
				signer = newMetadataCodec(t, alternateMetadataSigningKey)
			case "expired":
				metadata.ExpiresAt = metadataRequestTime()
			case "missing expiry":
				metadata.ExpiresAt = time.Time{}
			case "invalid client IP":
				metadata.ClientIP = "invalid"
			case "no hops":
				metadata.Hops = nil
			case "empty ingress ID":
				metadata.Hops[0].RequestID = ""
			}
			req := requestWithMetadata(t, signer, metadata)
			switch test.name {
			case "missing":
				req.Header.Del(proxy.HeaderFrontlineMeta)
			case "malformed":
				req.Header.Set(proxy.HeaderFrontlineMeta, "not-signed")
			case "duplicate":
				req.Header.Add(proxy.HeaderFrontlineMeta, req.Header.Get(proxy.HeaderFrontlineMeta))
			}
			req.Header.Set(proxy.HeaderRequestID, uid.New(uid.RequestPrefix))
			sess := newMetadataSession(t, req)
			want := sess.RequestID()
			if test.trustedID {
				want = ingressID
			}
			_, err := applyPeerMetadata(sess, codec, metadataRequestTime())
			require.NoError(t, err)
			require.Equal(t, want, sess.RequestID())
			require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
		})
	}
}

func TestRequestHops_IgnoresMalformedMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "client-controlled-value")

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresWrongSigningKey(t *testing.T) {
	t.Parallel()

	signer := newMetadataCodec(t, metadataSigningKey)
	verifier := newMetadataCodec(t, alternateMetadataSigningKey)
	req := requestWithMetadata(t, signer, validMetadata())

	hops, err := applyPeerMetadata(newMetadataSession(t, req), verifier, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMissingExpiry(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = time.Time{}
	req := requestWithMetadata(t, codec, metadata)

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresExpiredMetadata(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = metadataRequestTime().Add(-time.Minute)
	req := requestWithMetadata(t, codec, metadata)

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresMetadataAtExpiry(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.ExpiresAt = metadataRequestTime()
	req := requestWithMetadata(t, codec, metadata)

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresDuplicateHeaders(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	metadata := validMetadata()
	metadata.DebugTelemetryEnabled = true
	req := requestWithMetadata(t, codec, metadata)
	req.Header.Add(proxy.HeaderFrontlineMeta, req.Header.Get(proxy.HeaderFrontlineMeta))

	sess := newMetadataSession(t, req)
	hops, err := applyPeerMetadata(sess, codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.False(t, sess.DebugTelemetryEnabled())
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresEmptyHeader(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, "")

	hops, err := applyPeerMetadata(newMetadataSession(t, req), nil, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_IgnoresOversizedHeader(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, strings.Repeat("a", 4097))

	hops, err := applyPeerMetadata(newMetadataSession(t, req), codec, metadataRequestTime())
	require.NoError(t, err)
	require.Empty(t, hops)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_RejectsMetadataWithoutCodec(t *testing.T) {
	t.Parallel()

	codec := newMetadataCodec(t, metadataSigningKey)
	req := requestWithMetadata(t, codec, validMetadata())

	_, err := applyPeerMetadata(newMetadataSession(t, req), nil, metadataRequestTime())
	requireFrontlineMetadataError(t, err)
	require.Empty(t, req.Header.Values(proxy.HeaderFrontlineMeta))
}

func TestRequestHops_ClientIP(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		ip      string
		key     string
		expires time.Time
		want    string
	}{
		{"ipv4", "198.51.100.42", metadataSigningKey, metadataRequestTime().Add(time.Minute), "198.51.100.42"},
		{"ipv6", "2001:db8::42", metadataSigningKey, metadataRequestTime().Add(time.Minute), "2001:db8::42"},
		{"old peer", "", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"invalid ip", "attacker", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"zone", "fe80::1%eth0", metadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"wrong key", "198.51.100.42", alternateMetadataSigningKey, metadataRequestTime().Add(time.Minute), "10.1.2.3"},
		{"expired", "198.51.100.42", metadataSigningKey, metadataRequestTime(), "10.1.2.3"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			metadata := validMetadata()
			metadata.ClientIP = tt.ip
			metadata.ExpiresAt = tt.expires
			req := requestWithMetadata(t, newMetadataCodec(t, tt.key), metadata)
			req.RemoteAddr = "10.1.2.3:443"
			req.Header.Set("X-Forwarded-For", "203.0.113.77")
			sess := newMetadataSession(t, req)
			_, err := applyPeerMetadata(sess, newMetadataCodec(t, metadataSigningKey), metadataRequestTime())
			require.NoError(t, err)
			require.Equal(t, tt.want, sess.Location())
			require.Equal(t, "10.1.2.3:443", req.RemoteAddr)
			require.Empty(t, req.Header.Get(proxy.HeaderFrontlineMeta))
		})
	}
}

func TestRequestHops_DebugTelemetryEnabled(t *testing.T) {
	for _, test := range []struct {
		name         string
		debug        bool
		requestDebug bool
		key          string
		expires      time.Time
		ip           string
		want         bool
	}{
		{"enabled", true, false, metadataSigningKey, metadataRequestTime().Add(time.Minute), "", true},
		{"disabled overrides request header", false, true, metadataSigningKey, metadataRequestTime().Add(time.Minute), "", false},
		{"wrong key", true, false, alternateMetadataSigningKey, metadataRequestTime().Add(time.Minute), "", false},
		{"expired", true, false, metadataSigningKey, metadataRequestTime(), "", false},
		{"missing expiry", true, false, metadataSigningKey, time.Time{}, "", false},
		{"invalid IP", true, false, metadataSigningKey, metadataRequestTime().Add(time.Minute), "invalid", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := validMetadata()
			metadata.DebugTelemetryEnabled = test.debug
			metadata.ExpiresAt = test.expires
			metadata.ClientIP = test.ip
			req := requestWithMetadata(t, newMetadataCodec(t, test.key), metadata)
			if test.requestDebug {
				req.Header.Set("X-Unkey-Debug", "1")
			}
			sess := newMetadataSession(t, req)
			_, err := applyPeerMetadata(sess, newMetadataCodec(t, metadataSigningKey), metadataRequestTime())
			require.NoError(t, err)
			require.Equal(t, test.want, sess.DebugTelemetryEnabled())
			require.Empty(t, req.Header.Get(proxy.HeaderFrontlineMeta))
		})
	}
}

func TestGateway_DebugTelemetryResponse(t *testing.T) {
	upstreamHeaders := make(chan http.Header, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHeaders <- r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)
	codec := newMetadataCodec(t, metadataSigningKey)
	clk := clock.NewTestClock(metadataRequestTime())
	transports := proxy.NewTransportRegistry()
	t.Cleanup(transports.CloseIdleConnections)
	//nolint:exhaustruct
	service, err := proxy.New(proxy.Config{Clock: clk, Metadata: codec, UpstreamTransports: transports})
	require.NoError(t, err)
	srv, err := zen.New(zen.Config{})
	require.NoError(t, err)
	srv.RegisterRoute([]zen.Middleware{middleware.WithReservedHeaderStrip()},
		zen.NewRoute(http.MethodGet, "/", func(ctx context.Context, sess *zen.Session) error {
			if _, err := applyPeerMetadata(sess, codec, clk.Now()); err != nil {
				return err
			}
			ctx = proxy.WithRequestStartTime(ctx, clk.Now())
			return service.ForwardToInstance(ctx, sess, db.DeploymentsUpstreamProtocolHttp1,
				db.FindInstancesByDeploymentIDRow{Address: strings.TrimPrefix(upstream.URL, "http://")})
		}))

	for _, test := range []struct {
		name      string
		header    string
		peer      bool
		peerDebug bool
		want      bool
	}{
		{"missing", "", false, false, false},
		{"false", "false", false, false, false},
		{"invalid", "invalid", false, false, false},
		{"true", "true", false, false, true},
		{"verified peer", "", true, true, true},
		{"peer disabled overrides raw header", "true", true, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.header != "" {
				req.Header.Set("X-Unkey-Debug", test.header)
			}
			if test.peer {
				metadata := validMetadata()
				metadata.DebugTelemetryEnabled = test.peerDebug
				token, err := codec.Marshal(metadata)
				require.NoError(t, err)
				req.Header.Set(proxy.HeaderFrontlineMeta, token)
			}
			recorder := httptest.NewRecorder()
			srv.Mux().ServeHTTP(recorder, req)
			response := recorder.Result()
			t.Cleanup(func() { require.NoError(t, response.Body.Close()) })
			require.Equal(t, http.StatusNoContent, response.StatusCode)
			headers := <-upstreamHeaders
			require.Empty(t, headers.Values("X-Unkey-Debug"))
			require.Empty(t, headers.Values(proxy.HeaderFrontlineMeta))
			values := response.Header.Values(timing.HeaderName)
			if test.want {
				require.Len(t, values, 1)
				entries, err := timing.ParseEntries(values[0])
				require.NoError(t, err)
				operations := map[string]bool{}
				for _, entry := range entries {
					operations[entry.Name] = true
				}
				require.Equal(t, map[string]bool{"frontline_routing": true, "frontline": true, "total": true}, operations)
			} else {
				require.Empty(t, values)
			}
		})
	}
}

func newMetadataCodec(t *testing.T, signingKey string) *meta.Codec {
	t.Helper()

	codec, err := meta.New(signingKey)
	require.NoError(t, err)
	return codec
}

func newMetadataSession(t *testing.T, req *http.Request) *zen.Session {
	t.Helper()
	sess := &zen.Session{}
	require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
	return sess
}

func requestWithMetadata(t *testing.T, codec *meta.Codec, metadata *meta.Metadata) *http.Request {
	t.Helper()

	token, err := codec.Marshal(metadata)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "https://example.com", nil)
	req.Header.Set(proxy.HeaderFrontlineMeta, token)
	return req
}

func validMetadata() *meta.Metadata {
	return &meta.Metadata{
		Claims: paseto.Claims{
			ExpiresAt: metadataRequestTime().Add(time.Hour),
		},
		Hops: []meta.Hop{
			{
				Region:         "aws::us-east-1",
				RegionPlatform: "us-east-1.aws",
				RequestID:      "req_123",
				FrontlineID:    "frontline_123",
				TimeUnixMilli:  1_777_000_000_123,
			},
		},
	}
}

func metadataRequestTime() time.Time {
	return time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
}

func requireFrontlineMetadataError(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InternalServerError.URN(), code)
}
