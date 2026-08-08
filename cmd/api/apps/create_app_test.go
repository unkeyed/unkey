package apps

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestCreateApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2AppsCreateAppRequestBody
	}{{"all fields", "apps create-app --project=payments --name=Payments --slug=payments-api", openapi.V2AppsCreateAppRequestBody{Project: "payments", Name: "Payments", Slug: "payments-api"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequest[openapi.V2AppsCreateAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
