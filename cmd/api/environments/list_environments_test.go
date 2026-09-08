package environments

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestListEnvironments(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2EnvironmentsListEnvironmentsRequestBody
	}{{"request", "environments list-environments --project=x --app=x", openapi.V2EnvironmentsListEnvironmentsRequestBody{Project: "x", App: "x"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequestWithData[openapi.V2EnvironmentsListEnvironmentsRequestBody](t, Cmd(), tt.args, []any{})
			require.Equal(t, tt.want, got)
		})
	}
}
