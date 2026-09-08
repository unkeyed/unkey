package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListIdentities(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2IdentitiesListIdentitiesRequestBody
	}{
		{
			name: "minimal",
			args: "identities list-identities",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit: new(100),
			},
		},
		{
			name: "with limit",
			args: "identities list-identities --limit=50",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit: new(50),
			},
		},
		{
			name: "with cursor",
			args: "identities list-identities --cursor=cursor_eyJrZXkiOiJrZXlfMTIzNCJ9",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit:  new(100),
				Cursor: new("cursor_eyJrZXkiOiJrZXlfMTIzNCJ9"),
			},
		},
		{
			name: "with search",
			args: "identities list-identities --search=user_123",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit:  new(100),
				Search: new("user_123"),
			},
		},
		{
			name: "with all flags",
			args: "identities list-identities --limit=25 --cursor=cursor_123 --search=user_123",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit:  new(25),
				Cursor: new("cursor_123"),
				Search: new("user_123"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequestWithData[openapi.V2IdentitiesListIdentitiesRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, req)
		})
	}
}
