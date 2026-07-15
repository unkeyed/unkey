package proxy

import (
	"context"
	"net"
	"net/url"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fault"
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
