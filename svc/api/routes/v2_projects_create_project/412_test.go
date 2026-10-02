package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_create_project"
)

func TestCreateProjectWithoutComputePlan(t *testing.T) {
	h := testutil.NewHarness(t)
	const internalMessage = "workspace ws_secret has no Compute plan"
	route := &handler.Handler{
		CtrlClient: &testutil.MockProjectClient{
			CreateProjectFunc: func(context.Context, *ctrlv1.CreateProjectRequest) (*ctrlv1.CreateProjectResponse, error) {
				return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(internalMessage))
			},
		},
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "project.*.create_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, headers, handler.Request{
		Name: "Payments Service",
		Slug: "payments-service",
	})

	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, deploygate.MsgNoComputePlan, res.Body.Error.Detail)
	require.NotContains(t, res.RawBody, internalMessage)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
}
