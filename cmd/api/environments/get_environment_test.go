package environments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestGetEnvironment(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsGetEnvironmentRequestBody
	}{{"request", "environments get-environment --project=x --app=x --environment=x", openapi.V2EnvironmentsGetEnvironmentRequestBody{Project: "x", App: "x", Environment: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2EnvironmentsGetEnvironmentRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}
