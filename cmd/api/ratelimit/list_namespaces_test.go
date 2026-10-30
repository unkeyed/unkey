package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListNamespaces(t *testing.T) {
	cursor := uid.New(uid.RatelimitNamespacePrefix)
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitListNamespacesRequestBody
	}{
		{
			name: "omitted flags use SDK page size default",
			args: "ratelimit list-namespaces",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit: new(100),
			},
		},
		{
			name: "custom page size",
			args: "ratelimit list-namespaces --limit=20",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit: new(20),
			},
		},
		{
			name: "pagination cursor",
			args: "ratelimit list-namespaces --cursor=" + cursor,
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit:  new(100),
				Cursor: new(cursor),
			},
		},
		{
			name: "search filter",
			args: "ratelimit list-namespaces --search=email",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit:  new(100),
				Search: new("email"),
			},
		},
		{
			name: "explicit empty search is retained",
			args: "ratelimit list-namespaces --search=",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit:  new(100),
				Search: new(""),
			},
		},
		{
			name: "explicit zero is preserved for API validation",
			args: "ratelimit list-namespaces --limit=0",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit: new(0),
			},
		},
		{
			name: "explicit empty cursor is preserved for API validation",
			args: "ratelimit list-namespaces --cursor=",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit:  new(100),
				Cursor: new(""),
			},
		},
		{
			name: "all flags",
			args: "ratelimit list-namespaces --limit=20 --cursor=" + cursor + " --search=email",
			want: openapi.V2RatelimitListNamespacesRequestBody{
				Limit:  new(20),
				Cursor: new(cursor),
				Search: new("email"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2RatelimitListNamespacesRequestBody](t, Cmd(), tt.args, []openapi.RatelimitNamespace{})
			require.Equal(t, tt.want, got)
		})
	}
}
