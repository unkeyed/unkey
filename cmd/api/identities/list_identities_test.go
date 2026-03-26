package identities

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
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
				Limit: ptr.P(100),
			},
		},
		{
			name: "with limit",
			args: "identities list-identities --limit=50",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit: ptr.P(50),
			},
		},
		{
			name: "with cursor",
			args: "identities list-identities --cursor=cursor_eyJrZXkiOiJrZXlfMTIzNCJ9",
			want: openapi.V2IdentitiesListIdentitiesRequestBody{
				Limit:  ptr.P(100),
				Cursor: ptr.P("cursor_eyJrZXkiOiJrZXlfMTIzNCJ9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2IdentitiesListIdentitiesRequestBody](t, Cmd(), tt.args, `{"meta":{"requestId":"test"},"data":[]}`)
			require.Equal(t, tt.want, req)
		})
	}
}
