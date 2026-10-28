package workspace

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateFlag(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			want := openapi.V2WorkspaceUpdateFlagRequestBody{Slug: "example_feature", Enabled: enabled}
			body, err := json.Marshal(want)
			require.NoError(t, err)
			for _, args := range []string{
				fmt.Sprintf("workspace update-flag --slug=example_feature --enabled=%t", enabled),
				fmt.Sprintf("workspace update-flag --body='%s'", body),
			} {
				got := testutil.CaptureRequest[openapi.V2WorkspaceUpdateFlagRequestBody](t, Cmd(), args)
				require.Equal(t, want, got)
			}
		})
	}
}
