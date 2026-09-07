package deploymentgc

import (
	"context"
	"errors"
	"testing"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/depotcache"
	"google.golang.org/protobuf/proto"
)

type cacheProjectClient struct {
	corev1connect.ProjectServiceClient
	project   *corev1.Project
	update    *corev1.UpdateProjectRequest
	updateErr error
}

func (c *cacheProjectClient) GetProject(context.Context, *connect.Request[corev1.GetProjectRequest]) (*connect.Response[corev1.GetProjectResponse], error) {
	return connect.NewResponse(&corev1.GetProjectResponse{Project: proto.CloneOf(c.project)}), nil
}

func (c *cacheProjectClient) UpdateProject(_ context.Context, req *connect.Request[corev1.UpdateProjectRequest]) (*connect.Response[corev1.UpdateProjectResponse], error) {
	c.update = req.Msg
	return connect.NewResponse(&corev1.UpdateProjectResponse{}), c.updateErr
}

func TestCacheRetentionPreservesStorageAndProjectSettings(t *testing.T) {
	projects := &cacheProjectClient{project: &corev1.Project{
		ProjectId:   "build",
		Name:        "production-proj_test",
		RegionId:    "us-east-1",
		CachePolicy: &corev1.CachePolicy{KeepDays: 7, KeepGb: 25},
	}}
	client := &depotClient{projects: projects}
	changed, err := client.SetCacheRetention(t.Context(), "build", "production-proj_test", depotcache.RetentionDays)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, int32(3), projects.update.CachePolicy.GetKeepDays())
	require.Equal(t, int32(25), projects.update.CachePolicy.GetKeepGb())
	require.Nil(t, projects.update.Name)
	require.Nil(t, projects.update.RegionId)
	require.Nil(t, projects.update.Hardware)

	projects.update = nil
	projects.project.CachePolicy.KeepDays = depotcache.RetentionDays
	changed, err = client.SetCacheRetention(t.Context(), "build", "production-proj_test", depotcache.RetentionDays)
	require.NoError(t, err)
	require.False(t, changed)
	require.Nil(t, projects.update)

	projects.project.CachePolicy.KeepDays = 7
	projects.project.Name = "another-owner"
	changed, err = client.SetCacheRetention(t.Context(), "build", "production-proj_test", depotcache.RetentionDays)
	require.NoError(t, err)
	require.False(t, changed)
	require.Nil(t, projects.update)

	projects.project.Name = "production-proj_test"
	projects.updateErr = connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported retention"))
	changed, err = client.SetCacheRetention(t.Context(), "build", "production-proj_test", depotcache.RetentionDays)
	require.Error(t, err)
	require.False(t, changed)
}
