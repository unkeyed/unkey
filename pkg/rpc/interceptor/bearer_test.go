package interceptor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
)

func TestBearerAuthStaticAndExclusiveSources(t *testing.T) {
	for _, tc := range []struct {
		name, token, file string
		valid             bool
	}{
		{"static", "static-token", "", true},
		{"static-preserves-whitespace", " static-token ", "", true},
		{"neither", "", "", false},
		{"both", "static-token", filepath.Join(t.TempDir(), "missing"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			nextErr := errors.New("downstream failure")
			call := interceptor.NewBearerAuth(tc.token, tc.file).WrapUnary(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				called = true
				require.Equal(t, []string{"Bearer " + tc.token}, req.Header().Values("Authorization"))
				return nil, nextErr
			})
			req := connect.NewRequest(&vaultv1.EncryptRequest{})
			req.Header().Add("Authorization", "stale")
			req.Header().Add("Authorization", "duplicate")
			_, err := call(t.Context(), req)
			require.Equal(t, tc.valid, called)
			if tc.valid {
				require.ErrorIs(t, err, nextErr)
			} else {
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
			}
		})
	}
}

func TestBearerAuthFileRotationFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	calls := 0
	header := ""
	call := interceptor.NewBearerAuth("", path).WrapUnary(func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		calls++
		header = req.Header().Get("Authorization")
		return connect.NewResponse(&vaultv1.EncryptResponse{}), nil
	})
	req := connect.NewRequest(&vaultv1.EncryptRequest{})
	_, err := call(t.Context(), req)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Zero(t, calls)

	for _, tc := range []struct {
		name, contents, token string
	}{
		{"initial", " \tfirst-token\r\n", "first-token"},
		{"rotated", "second-token", "second-token"},
		{"empty", "", ""},
		{"whitespace", " \t\r\n", ""},
		{"space", "secret token", ""},
		{"tab", "secret\ttoken", ""},
		{"carriage-return", "secret\rtoken", ""},
		{"newline", "secret\ntoken", ""},
		{"size-limit", strings.Repeat("a", 16*1024), strings.Repeat("a", 16*1024)},
		{"oversize", strings.Repeat("a", 16*1024+1), ""},
		{"oversize-padding", "token" + strings.Repeat(" ", 16*1024), ""},
		{"recovered", "third-token\n", "third-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(path+".next", []byte(tc.contents), 0600))
			require.NoError(t, os.Rename(path+".next", path))
			before := calls
			_, err := call(t.Context(), req)
			if tc.token == "" {
				require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
				require.Equal(t, before, calls)
				if strings.TrimSpace(tc.contents) != "" {
					require.NotContains(t, err.Error(), tc.contents)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, before+1, calls)
			require.Equal(t, "Bearer "+tc.token, header)
		})
	}

	before := calls
	require.NoError(t, os.Remove(path))
	_, err = call(t.Context(), req)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Equal(t, before, calls)
	require.NoError(t, os.Mkdir(path, 0700))
	_, err = call(t.Context(), req)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	require.Equal(t, before, calls)
}
