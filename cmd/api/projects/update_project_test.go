package projects

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestUpdateProject(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2ProjectsUpdateProjectRequestBody
	}{{"omits optional fields", "projects update-project --project=proj_1", openapi.V2ProjectsUpdateProjectRequestBody{Project: "proj_1"}}, {"all options including false boolean", "projects update-project --project=proj_1 --slug=pay --name=Payments --delete-protection=false", openapi.V2ProjectsUpdateProjectRequestBody{Project: "proj_1", Slug: ptr.P("pay"), Name: ptr.P("Payments"), DeleteProtection: ptr.P(false)}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[openapi.V2ProjectsUpdateProjectRequestBody](t, Cmd(), tt.args))
		})
	}
}
