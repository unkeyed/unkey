package apps

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestDeleteApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2AppsDeleteAppRequestBody
	}{{"by id", "apps delete-app --project=payments --app=app_123", openapi.V2AppsDeleteAppRequestBody{Project: "payments", App: "app_123"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, captureAcceptedRequest[openapi.V2AppsDeleteAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
