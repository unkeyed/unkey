package middleware

import (
	"context"
	"errors"
	"fmt"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsClientGone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "context canceled", err: context.Canceled, want: true},
		{name: "wrapped context canceled", err: fmt.Errorf("send: %w", context.Canceled), want: true},
		{name: "EPIPE", err: syscall.EPIPE, want: true},
		{name: "ECONNRESET", err: syscall.ECONNRESET, want: true},
		{name: "http2 stream closed", err: errors.New("failed to send bytes: http2: stream closed"), want: true},
		{name: "client disconnected", err: errors.New("failed to send bytes: client disconnected"), want: true},
		{name: "broken pipe text", err: errors.New("write tcp: broken pipe"), want: true},
		{name: "connection reset by peer text", err: errors.New("read tcp: connection reset by peer"), want: true},
		{name: "unrelated error", err: errors.New("json marshal failed"), want: false},
		{name: "context deadline exceeded is server-side", err: context.DeadlineExceeded, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, isClientGone(tt.err))
		})
	}
}
