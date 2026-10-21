package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

func TestIsDialError(t *testing.T) {
	t.Parallel()

	dialOp := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	writeOp := &net.OpError{Op: "write", Net: "tcp", Err: syscall.ECONNRESET}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "dial OpError",
			err:  dialOp,
			want: true,
		},
		{
			name: "dial OpError wrapped in url.Error",
			err:  &url.Error{Op: "Get", URL: "http://example.com", Err: dialOp},
			want: true,
		},
		{
			name: "dial OpError wrapped in fault.Wrap",
			err:  fault.Wrap(dialOp, fault.Internal("upstream dial failed")),
			want: true,
		},
		{
			name: "DNSError",
			err:  &net.DNSError{Err: "no such host", Name: "example.com", IsNotFound: true},
			want: true,
		},
		{
			name: "DNSError wrapped in url.Error",
			err: &url.Error{
				Op:  "Get",
				URL: "http://example.com",
				Err: &net.DNSError{Err: "no such host", Name: "example.com", IsNotFound: true},
			},
			want: true,
		},
		{
			name: "plain ECONNREFUSED without dial OpError",
			err:  syscall.ECONNREFUSED,
			want: false,
		},
		{
			name: "context.Canceled",
			err:  context.Canceled,
			want: false,
		},
		{
			name: "context.DeadlineExceeded",
			err:  context.DeadlineExceeded,
			want: false,
		},
		{
			name: "mid-stream write OpError",
			err:  writeOp,
			want: false,
		},
		{
			name: "mid-stream write OpError wrapped in url.Error",
			err:  &url.Error{Op: "Post", URL: "http://example.com", Err: writeOp},
			want: false,
		},
		{
			name: "read OpError",
			err:  &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, IsDialError(tt.err))
		})
	}
}

func TestForwardErrorsAttributeInstanceAndPeerFailures(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		err          error
		instanceCode codes.URN
		peerCode     codes.URN
	}{
		{"refused", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, codes.Frontline.Proxy.BadGateway.URN(), codes.Frontline.Internal.InternalServerError.URN()},
		{"timeout", context.DeadlineExceeded, codes.Frontline.Proxy.GatewayTimeout.URN(), codes.Frontline.Internal.InternalServerError.URN()},
		{"unreachable", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EHOSTUNREACH}, codes.Frontline.Proxy.ServiceUnavailable.URN(), codes.Frontline.Internal.InternalServerError.URN()},
		{"client canceled", context.Canceled, codes.User.BadRequest.ClientClosedRequest.URN(), codes.User.BadRequest.ClientClosedRequest.URN()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			transport := &http.Transport{}
			failing := roundTripperFunc(func(*http.Request) (*http.Response, error) {
				return nil, tc.err
			})
			transport.RegisterProtocol("https", failing)
			t.Cleanup(transport.CloseIdleConnections)
			metadata, err := meta.New(testMetadataSigningKey)
			require.NoError(t, err)
			svc, err := New(Config{
				InstanceID: uid.New(uid.FrontlinePrefix), Platform: "aws", Region: "us-east-1",
				ApexDomain: "example.com", Clock: clock.New(), Metadata: metadata, Transport: transport,
				UpstreamTransports: &TransportRegistry{fallback: failing},
			})
			require.NoError(t, err)
			for _, destination := range []proxyDestination{destinationInstance, destinationFrontline} {
				t.Run(string(destination), func(t *testing.T) {
					sess := newProxySession(t, httptest.NewRequest(http.MethodGet, "https://app.example.com", nil))
					var forwardErr error
					want := tc.instanceCode
					if destination == destinationInstance {
						forwardErr = svc.ForwardToInstance(t.Context(), sess, db.DeploymentsUpstreamProtocolHttp1, db.FindInstancesByDeploymentIDRow{
							ID: uid.New(uid.InstancePrefix), Address: "instance.internal:8080",
						})
					} else {
						want = tc.peerCode
						forwardErr = svc.ForwardToRegion(t.Context(), sess, "us-west-2.aws", nil)
					}
					require.ErrorIs(t, forwardErr, tc.err)
					code, ok := fault.GetCode(forwardErr)
					require.True(t, ok)
					require.Equal(t, want, code)
				})
			}
		})
	}
}
