package apps

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
	"testing"
)

func TestCreateApp(t *testing.T) {
	tests := []struct {
		name, args string
		want       components.V2AppsCreateAppRequestBody
	}{{"required fields", "apps create-app --project=payments --name=Payments --slug=payments-api", components.V2AppsCreateAppRequestBody{Project: "payments", Name: "Payments", Slug: "payments-api"}}, {"with git", `apps create-app --project=payments --name=Payments --slug=payments-api --git={"repository":"unkeyed/api","defaultBranch":"main"}`, components.V2AppsCreateAppRequestBody{Project: "payments", Name: "Payments", Slug: "payments-api", Git: &components.AppGitCreateInput{Repository: "unkeyed/api", DefaultBranch: ptr.P("main")}}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[components.V2AppsCreateAppRequestBody](t, Cmd(), tt.args))
		})
	}
}
