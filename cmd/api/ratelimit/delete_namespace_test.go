package ratelimit

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteNamespace(t *testing.T) {
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	tests := []struct {
		name string
		args string
		body bool
		want openapi.V2RatelimitDeleteNamespaceRequestBody
	}{
		{
			name: "delete by ID",
			args: "ratelimit delete-namespace --namespace=" + namespaceID,
			want: openapi.V2RatelimitDeleteNamespaceRequestBody{
				Namespace: namespaceID,
			},
		},
		{
			name: "delete by name",
			args: "ratelimit delete-namespace --namespace=email.outbound",
			want: openapi.V2RatelimitDeleteNamespaceRequestBody{
				Namespace: "email.outbound",
			},
		},
		{
			name: "body selects the namespace to delete",
			body: true,
			want: openapi.V2RatelimitDeleteNamespaceRequestBody{
				Namespace: namespaceID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args
			if tt.body {
				body, err := json.Marshal(tt.want)
				require.NoError(t, err)
				args = "ratelimit delete-namespace --body='" + string(body) + "'"
			}
			got := testutil.CaptureRequest[openapi.V2RatelimitDeleteNamespaceRequestBody](t, Cmd(), args+" --yes")
			require.Equal(t, tt.want, got)
		})
	}
}
