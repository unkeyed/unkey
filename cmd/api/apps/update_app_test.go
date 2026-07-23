package apps

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestUpdateApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2AppsUpdateAppRequestBody
	}{{"omits optional fields", "apps update-app --project=payments --app=app_1", openapi.V2AppsUpdateAppRequestBody{Project: "payments", App: "app_1"}}, {"all options including false boolean", "apps update-app --project=payments --app=app_1 --name=Pay --slug=pay --default-branch=trunk --delete-protection=false", openapi.V2AppsUpdateAppRequestBody{Project: "payments", App: "app_1", Name: ptr.P("Pay"), Slug: ptr.P("pay"), DefaultBranch: ptr.P("trunk"), DeleteProtection: ptr.P(false)}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequest[openapi.V2AppsUpdateAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
