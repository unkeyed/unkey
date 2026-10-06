package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen/validation"
)

const dashboardCreateAgentKeyBody = `{
  "name": "Agent",
  "permissions": [
    "unkey:v1:ws_123:projects/*/keyspaces/*#read",
    "unkey:v1:ws_123:projects/*/keyspaces/*#write",
    "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#read",
    "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#write",
    "unkey:v1:ws_123:projects/*/keyspaces/*/keys/*#verify"
  ]
}`

func TestCreateAgentKeyOpenAPIAcceptsDashboardBody(t *testing.T) {
	t.Parallel()

	validator, err := validation.New()
	require.NoError(t, err)

	for _, target := range []string{
		"https://api.unkey.com/v2/rootKeys.createAgentKey",
		"http://localhost:7070/v2/rootKeys.createAgentKey",
		"http://api.unkey.local/v2/rootKeys.createAgentKey",
	} {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(dashboardCreateAgentKeyBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-token")

			got, valid := validator.Validate(t.Context(), req)
			require.Truef(t, valid, "status=%d title=%q type=%q detail=%q errors=%+v", got.Error.Status, got.Error.Title, got.Error.Type, got.Error.Detail, got.Error.Errors)
		})
	}
}
