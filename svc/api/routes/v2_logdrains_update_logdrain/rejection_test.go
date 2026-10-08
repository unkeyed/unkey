package logdrains_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestUpdateRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"missing ID", `{"name":"Changed"}`},
		{"blank name", `{"logdrainId":"ID","name":""}`},
		{"whitespace name", `{"logdrainId":"ID","name":" \t\n "}`},
		{"zero batch size", `{"logdrainId":"ID","name":"Changed","batchSize":0}`},
		{"negative batch size", `{"logdrainId":"ID","name":"Changed","batchSize":-1}`},
		{"overflow batch size", `{"logdrainId":"ID","name":"Changed","batchSize":4294967296}`},
		{"malformed JSON", `{"logdrainId":"ID","name":"Changed",`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, route, id, config := seedUpdateDrain(t)
			key := h.CreateRootKey(h.Resources().UserWorkspace.ID, urn.New().Workspace(h.Resources().UserWorkspace.ID).Logdrain(id).String()+"#write")
			req := httptest.NewRequest(route.Method(), route.Path(), strings.NewReader(strings.ReplaceAll(tc.body, "ID", id)))
			req.Header = updateHeaders(key)
			res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "%s", res.RawBody)
			require.Equal(t, 400, res.Body.Error.Status)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			assertUpdateState(t, h, id, "Original", config, false)
		})
	}
}

func TestUpdateMissingDrainDoesNotMutateExistingDrain(t *testing.T) {
	h, route, id, config := seedUpdateDrain(t)
	missingID := uid.New("ld")
	key := h.CreateRootKey(h.Resources().UserWorkspace.ID, urn.New().Workspace(h.Resources().UserWorkspace.ID).Logdrain(missingID).String()+"#write")
	req := httptest.NewRequest(route.Method(), route.Path(), strings.NewReader(`{"logdrainId":"`+missingID+`","name":"Changed"}`))
	req.Header = updateHeaders(key)
	res := testutil.CallRaw[openapi.NotFoundErrorResponse](h, req)
	require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/logdrain_not_found", res.Body.Error.Type)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	assertUpdateState(t, h, id, "Original", config, false)
	require.Empty(t, h.FindAuditLogsByTargetID(t.Context(), t, missingID))
}
