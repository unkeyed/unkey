package deploymentgc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/build/v1/buildv1connect"
	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	buildv1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/build/v1"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DepotImage is the registry identity needed for reconciliation.
type DepotImage struct {
	Tag      string
	Digest   string
	PushedAt time.Time
}

// DepotProject is the build-project identity needed for reconciliation.
type DepotProject struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// Depot is the narrow subset of the Depot API used by garbage collection.
type Depot interface {
	// ListImages returns one page of images and the token for the next page.
	ListImages(ctx context.Context, registryProjectID, pageToken string) ([]DepotImage, string, error)
	// DeleteImage deletes a tag. A missing tag must count as success.
	DeleteImage(ctx context.Context, registryProjectID, tag string) error
	// ListProjects returns one page of projects and the token for the next page.
	ListProjects(ctx context.Context, pageToken string) ([]DepotProject, string, error)
	// GetProject returns exists as false when the project does not exist.
	GetProject(ctx context.Context, projectID string) (DepotProject, bool, error)
	// DeleteProject deletes a project. A missing project must count as success.
	DeleteProject(ctx context.Context, projectID string) error
}

type depotClient struct {
	registry buildv1connect.RegistryServiceClient
	projects corev1connect.ProjectServiceClient
}

// ParseRegistryProjectID extracts Depot's registry project ID from its
// configured repository, for example registry.depot.dev/abc123 -> abc123.
func ParseRegistryProjectID(repository string) (string, error) {
	const depotRegistryPrefix = "registry.depot.dev/"
	projectID, ok := strings.CutPrefix(strings.TrimSuffix(repository, "/"), depotRegistryPrefix)
	if !ok || projectID == "" || strings.Contains(projectID, "/") {
		return "", fmt.Errorf("depot registry repository must match %s<project-id>", depotRegistryPrefix)
	}
	return projectID, nil
}

// NewDepotClient creates an authenticated Depot API client.
func NewDepotClient(apiURL, token string) Depot {
	auth := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
	options := []connect.ClientOption{connect.WithInterceptors(auth)}
	return &depotClient{
		registry: buildv1connect.NewRegistryServiceClient(&http.Client{}, apiURL, options...),
		projects: corev1connect.NewProjectServiceClient(&http.Client{}, apiURL, options...),
	}
}

func (c *depotClient) ListImages(ctx context.Context, registryProjectID, pageToken string) ([]DepotImage, string, error) {
	pageSize := int32(500)
	request := &buildv1.ListImagesRequest{ProjectId: registryProjectID, PageSize: &pageSize, PageToken: nil}
	if pageToken != "" {
		request.PageToken = &pageToken
	}
	response, err := c.registry.ListImages(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, "", err
	}
	images := make([]DepotImage, 0, len(response.Msg.GetImages()))
	for _, image := range response.Msg.GetImages() {
		images = append(images, DepotImage{
			Tag:      image.GetTag(),
			Digest:   image.GetDigest(),
			PushedAt: validTime(image.GetPushedAt()),
		})
	}
	return images, response.Msg.GetNextPageToken(), nil
}

func (c *depotClient) DeleteImage(ctx context.Context, registryProjectID, tag string) error {
	_, err := c.registry.DeleteImage(ctx, connect.NewRequest(&buildv1.DeleteImageRequest{
		ProjectId: registryProjectID,
		ImageTags: []string{tag},
	}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil
	}
	return err
}

func (c *depotClient) ListProjects(ctx context.Context, pageToken string) ([]DepotProject, string, error) {
	pageSize := int32(500)
	request := &corev1.ListProjectsRequest{RegionId: nil, PageSize: &pageSize, PageToken: nil}
	if pageToken != "" {
		request.PageToken = &pageToken
	}
	response, err := c.projects.ListProjects(ctx, connect.NewRequest(request))
	if err != nil {
		return nil, "", err
	}
	projects := make([]DepotProject, 0, len(response.Msg.GetProjects()))
	for _, project := range response.Msg.GetProjects() {
		projects = append(projects, DepotProject{
			ID:        project.GetProjectId(),
			Name:      project.GetName(),
			CreatedAt: validTime(project.GetCreatedAt()),
		})
	}
	return projects, response.Msg.GetNextPageToken(), nil
}

func (c *depotClient) GetProject(ctx context.Context, projectID string) (DepotProject, bool, error) {
	response, err := c.projects.GetProject(ctx, connect.NewRequest(&corev1.GetProjectRequest{ProjectId: projectID}))
	var noProject DepotProject
	if connect.CodeOf(err) == connect.CodeNotFound {
		return noProject, false, nil
	}
	if err != nil {
		return noProject, false, err
	}
	project := response.Msg.GetProject()
	return DepotProject{
		ID:        project.GetProjectId(),
		Name:      project.GetName(),
		CreatedAt: validTime(project.GetCreatedAt()),
	}, true, nil
}

func (c *depotClient) DeleteProject(ctx context.Context, projectID string) error {
	_, err := c.projects.DeleteProject(ctx, connect.NewRequest(&corev1.DeleteProjectRequest{ProjectId: projectID}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil
	}
	return err
}

func validTime(timestamp *timestamppb.Timestamp) time.Time {
	if timestamp == nil || timestamp.CheckValid() != nil {
		return time.Time{}
	}
	return timestamp.AsTime()
}
