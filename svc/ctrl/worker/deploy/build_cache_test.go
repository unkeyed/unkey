package deploy

import (
	"context"
	"net/http/httptest"
	"testing"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type cacheCreationDB struct {
	db.Database
	depotProjectID string
}

func (d *cacheCreationDB) FindProjectById(context.Context, string) (db.Project, error) {
	return db.Project{}, nil
}

func (d *cacheCreationDB) UpdateProjectDepotID(_ context.Context, params db.UpdateProjectDepotIDParams) error {
	d.depotProjectID = params.DepotProjectID.String
	return nil
}

type cacheCreationServer struct {
	corev1connect.UnimplementedProjectServiceHandler
	request *corev1.CreateProjectRequest
}

func (s *cacheCreationServer) CreateProject(_ context.Context, req *connect.Request[corev1.CreateProjectRequest]) (*connect.Response[corev1.CreateProjectResponse], error) {
	s.request = req.Msg
	return connect.NewResponse(&corev1.CreateProjectResponse{Project: &corev1.Project{ProjectId: "depot-project"}}), nil
}

func TestNewDepotProjectUsesThreeDayLayerCache(t *testing.T) {
	projects := &cacheCreationServer{}
	_, handler := corev1connect.NewProjectServiceHandler(projects)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	database := &cacheCreationDB{}
	w := &Workflow{db: database, buildConfig: BuildConfig{Depot: DepotConfig{
		APIUrl: server.URL, ProjectPrefix: "production", ProjectRegion: "us-east-1",
	}}}
	id, err := w.getOrCreateDepotProject(t.Context(), "proj_test")
	require.NoError(t, err)
	require.Equal(t, "depot-project", id)
	require.Equal(t, id, database.depotProjectID)
	require.Equal(t, "production-proj_test", projects.request.GetName())
	require.Equal(t, int32(3), projects.request.GetCachePolicy().GetKeepDays())
	require.Equal(t, int32(25), projects.request.GetCachePolicy().GetKeepGb())
}
