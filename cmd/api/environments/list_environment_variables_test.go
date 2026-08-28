package environments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestListEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsListEnvironmentVariablesRequestBody
	}{{"request", "environments list-environment-variables --project=x --app=x --environment=x", openapi.V2EnvironmentsListEnvironmentVariablesRequestBody{Project: "x", App: "x", Environment: "x", Limit: ptr.P(100), Cursor: nil}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2EnvironmentsListEnvironmentVariablesRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, got)
		})
	}
}
