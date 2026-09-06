package apps

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestUpdateApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       components.V2AppsUpdateAppRequestBody
	}{{"omits optional fields", "apps update-app --project=payments --app=app_1", components.V2AppsUpdateAppRequestBody{Project: "payments", App: "app_1"}}, {"all options including false boolean", `apps update-app --project=payments --app=app_1 --name=Pay --slug=pay --git='{"repository":"unkeyed/api","defaultBranch":"trunk"}' --delete-protection=false`, components.V2AppsUpdateAppRequestBody{Project: "payments", App: "app_1", Name: new("Pay"), Slug: new("pay"), Git: map[bool]*components.AppGitUpdateInput{true: {Repository: new("unkeyed/api"), DefaultBranch: new("trunk")}}, DeleteProtection: new(false)}}, {"disconnect git", "apps update-app --project=payments --app=app_1 --git=null", components.V2AppsUpdateAppRequestBody{Project: "payments", App: "app_1", Git: map[bool]*components.AppGitUpdateInput{true: nil}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[components.V2AppsUpdateAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
