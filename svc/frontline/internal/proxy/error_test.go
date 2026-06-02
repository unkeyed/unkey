package proxy

import (
	"context"
	"net"
	"net/url"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/frontline/internal/publicerr"
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

// TestCategorizeProxyError pins the syscall → URN mapping. The Outcome
// each URN maps to lives in svc/frontline/internal/metrics/outcome.go;
// here we verify only that the syscall lands in the right URN.
//
// Two destinations are covered:
//   - destinationInstance — customer pod. upstream_* URNs.
//   - destinationFrontline — peer frontline. peer_frontline_* URNs.
//
// The peer-frontline rewrites are the most error-prone part of the
// refactor: a peer-frontline failure that was previously bucketed
// alongside customer-instance failures would silently land in the
// upstream_problem outcome and never page on-call.
func TestCategorizeProxyError(t *testing.T) {
	t.Parallel()

	dialTimeout := &net.OpError{Op: "dial", Net: "tcp", Err: &timeoutErr{}}
	readTimeout := &net.OpError{Op: "read", Net: "tcp", Err: &timeoutErr{}}
	connRefused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	connReset := &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}
	hostUnreachable := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EHOSTUNREACH}
	dnsNotFound := &net.DNSError{Err: "no such host", Name: "example.com", IsNotFound: true}
	dnsTimeout := &net.DNSError{Err: "timeout", Name: "example.com", IsTimeout: true}

	tests := []struct {
		name        string
		err         error
		destination string
		wantURN     codes.URN
	}{
		// ── Cross-destination URNs ─────────────────────────────────
		{
			name:        "client closed request (instance)",
			err:         context.Canceled,
			destination: destinationInstance,
			wantURN:     codes.User.BadRequest.ClientClosedRequest.URN(),
		},
		{
			name:        "client closed request (frontline)",
			err:         context.Canceled,
			destination: destinationFrontline,
			wantURN:     codes.User.BadRequest.ClientClosedRequest.URN(),
		},
		{
			// Instance path: a slow customer app tripping our outer
			// deadline is an upstream problem, not a frontline fault —
			// it must not page on-call.
			name:        "our outer deadline (instance)",
			err:         context.DeadlineExceeded,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
		},
		{
			// Peer-frontline path: a hop we own was too slow → fault.
			name:        "our outer deadline (frontline)",
			err:         context.DeadlineExceeded,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.GatewayDeadlineExceeded.URN(),
		},

		// ── Customer-instance path ─────────────────────────────────
		{
			name:        "ECONNREFUSED to instance",
			err:         connRefused,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.UpstreamConnectionRefused.URN(),
		},
		{
			name:        "ECONNRESET to instance",
			err:         connReset,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.UpstreamConnectionReset.URN(),
		},
		{
			name:        "EHOSTUNREACH to instance",
			err:         hostUnreachable,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.UpstreamHostUnreachable.URN(),
		},
		{
			name:        "dial timeout to instance",
			err:         dialTimeout,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.DialTimeout.URN(),
		},
		{
			name:        "response timeout from instance",
			err:         readTimeout,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.UpstreamResponseTimeout.URN(),
		},

		// ── Instance DNS: defensive branch ─────────────────────────
		// DNS shouldn't fire for instance dials (we dial by IP).
		// If it does, it's a frontline bug — collapse to
		// ProxyErrorUnclassified (frontline_fault outcome) so on-call
		// gets paged. Tests both DNS subtypes to lock the behavior.
		{
			name:        "DNS NXDOMAIN on instance path → unclassified",
			err:         dnsNotFound,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		},
		{
			name:        "DNS timeout on instance path → unclassified",
			err:         dnsTimeout,
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		},

		// ── Peer-frontline path (destination override) ─────────────
		{
			name:        "ECONNREFUSED to peer frontline",
			err:         connRefused,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineConnectionRefused.URN(),
		},
		{
			name:        "ECONNRESET to peer frontline",
			err:         connReset,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineConnectionReset.URN(),
		},
		{
			name:        "EHOSTUNREACH to peer frontline",
			err:         hostUnreachable,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineHostUnreachable.URN(),
		},
		{
			name:        "dial timeout to peer frontline",
			err:         dialTimeout,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineTimeout.URN(),
		},
		{
			name:        "DNS NXDOMAIN for peer frontline",
			err:         dnsNotFound,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineDNSNotFound.URN(),
		},
		{
			name:        "DNS timeout for peer frontline",
			err:         dnsTimeout,
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.PeerFrontlineDNSTimeout.URN(),
		},

		// ── Fallback: unclassified errors default to frontline_fault ─
		{
			name:        "unclassified error to instance",
			err:         fault.New("something weird"),
			destination: destinationInstance,
			wantURN:     codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		},
		{
			name:        "unclassified error to peer frontline",
			err:         fault.New("something weird"),
			destination: destinationFrontline,
			wantURN:     codes.Frontline.Proxy.ProxyErrorUnclassified.URN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotURN, _ := categorizeProxyError(tt.err, tt.destination)
			require.Equal(t, tt.wantURN, gotURN)
		})
	}
}

// TestCategorizeProxyError_NoTopologyLeak asserts the customer-facing
// message returned alongside each URN does not contain any of the
// forbidden substrings in publicerr.ForbiddenInPublicMessages. These
// messages flow to the client via fault.UserFacingMessage and override
// the catalog default in publicerr — they bypass the catalog's careful
// generic phrasing, so this is the only test that guards them.
//
// The error cases mirror TestCategorizeProxyError so adding a new
// classification branch there also covers it here.
func TestCategorizeProxyError_NoTopologyLeak(t *testing.T) {
	t.Parallel()

	dialTimeout := &net.OpError{Op: "dial", Net: "tcp", Err: &timeoutErr{}}
	readTimeout := &net.OpError{Op: "read", Net: "tcp", Err: &timeoutErr{}}
	connRefused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	connReset := &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}
	hostUnreachable := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.EHOSTUNREACH}
	dnsNotFound := &net.DNSError{Err: "no such host", Name: "example.com", IsNotFound: true}
	dnsTimeout := &net.DNSError{Err: "timeout", Name: "example.com", IsTimeout: true}

	errs := []error{
		context.Canceled,
		context.DeadlineExceeded,
		connRefused,
		connReset,
		hostUnreachable,
		dialTimeout,
		readTimeout,
		dnsNotFound,
		dnsTimeout,
		fault.New("something weird"),
	}
	destinations := []string{destinationInstance, destinationFrontline}

	for _, dest := range destinations {
		for _, err := range errs {
			_, msg := categorizeProxyError(err, dest)
			if bad := publicerr.ContainsForbidden(msg); bad != "" {
				t.Errorf("categorizeProxyError(%T, %q) returned message "+
					"that leaks forbidden substring %q:\n%s",
					err, dest, bad, msg)
			}
		}
	}
}

// timeoutErr satisfies net.Error.Timeout() == true. Used to drive
// net.OpError.Timeout() in the categorizeProxyError tests without
// depending on system-level deadline handling.
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
