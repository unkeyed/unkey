package apps

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestGetApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2AppsGetAppRequestBody
	}{{"by slug", "apps get-app --project=payments --app=payments-api", openapi.V2AppsGetAppRequestBody{Project: "payments", App: "payments-api"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[openapi.V2AppsGetAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
