package gdprerasure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// depotServer records the projects it is asked to delete and fails each
// request with err when set.
type depotServer struct {
	corev1connect.UnimplementedProjectServiceHandler
	err     error
	auth    []string
	deleted []string
}

func (s *depotServer) DeleteProject(_ context.Context, req *connect.Request[corev1.DeleteProjectRequest]) (*connect.Response[corev1.DeleteProjectResponse], error) {
	s.auth = append(s.auth, req.Header().Get("Authorization"))
	s.deleted = append(s.deleted, req.Msg.GetProjectId())
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(&corev1.DeleteProjectResponse{}), nil
}

// TestDeleteProjectTreatsMissingAsDeleted guarantees that a workspace erasure
// converges on Depot: a project an earlier run already deleted counts as done,
// while any other Depot failure surfaces so the step retries.
func TestDeleteProjectTreatsMissingAsDeleted(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		existed bool
		success bool
	}{
		{name: "deleted", err: nil, existed: true, success: true},
		{name: "already gone", err: connect.NewError(connect.CodeNotFound, nil), existed: false, success: true},
		{name: "failed", err: connect.NewError(connect.CodeInternal, nil), existed: false, success: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := &depotServer{err: tc.err}
			mux := http.NewServeMux()
			mux.Handle(corev1connect.NewProjectServiceHandler(fake))
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)

			projectID := uid.New(uid.TestPrefix)
			existed, err := NewDepot(server.URL, "depot_test").DeleteProject(t.Context(), projectID)
			require.Equal(t, []string{projectID}, fake.deleted)
			require.Equal(t, []string{"Bearer depot_test"}, fake.auth)
			require.Equal(t, tc.existed, existed)
			if tc.success {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
