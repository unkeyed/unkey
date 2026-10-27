package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_notifications_delete_destination"
)

type rejectingNotificationService struct {
	ctrlv1connect.UnimplementedNotificationServiceHandler
	code connect.Code
}

func (s rejectingNotificationService) DeleteDestination(context.Context, *connect.Request[ctrlv1.NotificationDeleteDestinationRequest]) (*connect.Response[ctrlv1.NotificationDeleteDestinationResponse], error) {
	return nil, connect.NewError(s.code, errors.New("internal: https://example.com/SECRET header=SECRET signingSecret=SECRET"))
}

func TestDeleteDestinationFailureIsActionableAndRedacted(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   connect.Code
		status int
		detail string
	}{
		{
			name:   "referenced destination",
			code:   connect.CodeFailedPrecondition,
			status: http.StatusPreconditionFailed,
			detail: "Remove this destination from all notification rules before deleting it.",
		},
		{
			name:   "internal failure",
			code:   connect.CodeInternal,
			status: http.StatusInternalServerError,
			detail: "Failed to delete notification destination.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := testutil.NewHarness(t)
			_, rpcHandler := ctrlv1connect.NewNotificationServiceHandler(rejectingNotificationService{code: tc.code})
			rpcServer := httptest.NewServer(rpcHandler)
			t.Cleanup(rpcServer.Close)
			route := &handler.Handler{Notifications: ctrlv1connect.NewNotificationServiceClient(rpcServer.Client(), rpcServer.URL)}
			h.Register(route)
			workspaceID := h.Resources().UserWorkspace.ID
			root := h.CreateRootKey(workspaceID, "unkey:v1:"+workspaceID+":notifications#write")
			res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, testutil.RootKeyHeaders(root), handler.Request{
				DestinationId: uid.New(uid.TestPrefix),
			})
			require.Equal(t, tc.status, res.Status, res.RawBody)
			require.Equal(t, tc.detail, res.Body.Error.Detail)
			require.Equal(t, tc.status, res.Body.Error.Status)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.NotContains(t, res.RawBody, "SECRET")
			require.NotContains(t, res.RawBody, "https://example.com")
			if tc.code == connect.CodeFailedPrecondition {
				require.Equal(t, "Precondition Failed", res.Body.Error.Title)
				require.Equal(t, codes.App.Precondition.PreconditionFailed.DocsURL(), res.Body.Error.Type)
			}
		})
	}
}
