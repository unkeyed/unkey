package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateNamespace(t *testing.T) {
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitUpdateNamespaceRequestBody
	}{
		{
			name: "rename by ID",
			args: "ratelimit update-namespace --namespace=" + namespaceID + " --name=email.outbound",
			want: openapi.V2RatelimitUpdateNamespaceRequestBody{
				Namespace: namespaceID,
				Name:      "email.outbound",
			},
		},
		{
			name: "rename by current name",
			args: "ratelimit update-namespace --namespace=email.sends --name=email.outbound",
			want: openapi.V2RatelimitUpdateNamespaceRequestBody{
				Namespace: "email.sends",
				Name:      "email.outbound",
			},
		},
		{
			name: "preserve case and spaces",
			args: `ratelimit update-namespace --namespace="Email Sends" --name="Email Outbound"`,
			want: openapi.V2RatelimitUpdateNamespaceRequestBody{
				Namespace: "Email Sends",
				Name:      "Email Outbound",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2RatelimitUpdateNamespaceRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}
