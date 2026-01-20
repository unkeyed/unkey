package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlBuildClient,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "project.*.generate_upload_url")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("project not found", func(t *testing.T) {
		req := handler.Request{
			ProjectId: uid.New(uid.ProjectPrefix), // Non-existent project ID
		}

		res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 400, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "The requested project does not exist or has been deleted.", res.Body.Error.Detail)
	})
}
