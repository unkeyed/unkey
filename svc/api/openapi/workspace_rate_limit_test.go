package openapi_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestWorkspaceRateLimitExemptionsFromSpec(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		annotation string
		exempt     bool
		invalid    bool
	}{
		{name: "explicit exemption", annotation: "x-unkey-workspace-rate-limit-exempt: true", exempt: true},
		{name: "explicit inclusion", annotation: "x-unkey-workspace-rate-limit-exempt: false"},
		{name: "no annotation"},
		{name: "idempotency is not an exemption", annotation: "x-unkey-idempotency: idempotent"},
		{name: "string is not a boolean", annotation: `x-unkey-workspace-rate-limit-exempt: "true"`, invalid: true},
		{name: "number is not a boolean", annotation: "x-unkey-workspace-rate-limit-exempt: 1", invalid: true},
		{name: "null is not a boolean", annotation: "x-unkey-workspace-rate-limit-exempt: null", invalid: true},
		{name: "yaml shorthand is not a boolean", annotation: "x-unkey-workspace-rate-limit-exempt: yes", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := fmt.Sprintf(`paths:
  /operation:
    summary: Shared path metadata
    parameters: []
    post:
      operationId: example
      %s
    get:
      operationId: other
`, tt.annotation)
			exemptions, err := openapi.WorkspaceRateLimitExemptionsFromSpec([]byte(spec))
			if tt.invalid {
				require.Error(t, err)
				require.False(t, exemptions.Contains(http.MethodPost, "/operation"))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.exempt, exemptions.Contains(http.MethodPost, "/operation"))
			require.False(t, exemptions.Contains(http.MethodGet, "/operation"))
			require.False(t, exemptions.Contains(http.MethodPost, "/other"))
		})
	}
}

func TestWorkspaceRateLimitExemptionsRejectMalformedSpec(t *testing.T) {
	t.Parallel()
	exemptions, err := openapi.WorkspaceRateLimitExemptionsFromSpec([]byte("paths: ["))
	require.Error(t, err)
	require.False(t, exemptions.Contains(http.MethodPost, "/operation"))
}

func TestWorkspaceRateLimitExemptionsZeroValueExemptsNothing(t *testing.T) {
	t.Parallel()
	var exemptions openapi.WorkspaceRateLimitExemptions
	require.False(t, exemptions.Contains(http.MethodPost, "/v2/deployments.listDeployments"))
}
