// Package depotclient wraps the Depot API surface the control plane uses
// for cleanup: registry tag enumeration/deletion
// (depot.registry.v1beta1.RegistryService) and project listing/deletion
// (depot.core.v1.ProjectService). Consumers depend on the narrow slice
// they need; [Client] (real) and [Noop] (Depot not configured) satisfy
// all of them.
package depotclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	corev1connect "buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	"buf.build/gen/go/depot/api/connectrpc/go/depot/registry/v1beta1/registryv1beta1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	registryv1beta1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/registry/v1beta1"
	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/pkg/assert"
)

// Image is one registry manifest with its tags.
type Image struct {
	Tags     []string  `json:"tags"`
	PushedAt time.Time `json:"pushedAt"`
}

// Project is one Depot build project.
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// API is the full Depot surface this package exposes. Consumers should
// declare the subset they need; this union exists so callers can hold one
// value that is either a [Client] or a [Noop].
type API interface {
	ListImages(ctx context.Context, repository string, page int32) (images []Image, hasMore bool, err error)
	DeleteTag(ctx context.Context, repository, tag string) error
	ListProjects(ctx context.Context, pageToken string) (projects []Project, nextPageToken string, err error)
	DeleteProject(ctx context.Context, projectID string) error
}

// listImagesPageSize is the page size requested from the registry API.
const listImagesPageSize = 100

// requestTimeout bounds Depot calls independently of Restate's retry budget.
const requestTimeout = 30 * time.Second

// Config holds the Depot API connection settings.
type Config struct {
	// APIUrl is the Depot API endpoint, e.g. "https://api.depot.dev".
	APIUrl string
	// Token is the Depot organization token (the registry password).
	Token string
}

// Client calls the Depot API.
type Client struct {
	registry registryv1beta1connect.RegistryServiceClient
	projects corev1connect.ProjectServiceClient
}

var _ API = (*Client)(nil)

// New constructs a Client.
func New(cfg Config) (*Client, error) {
	if err := assert.All(
		assert.NotEmpty(cfg.APIUrl, "APIUrl must not be empty"),
		assert.NotEmpty(cfg.Token, "Token must not be empty"),
	); err != nil {
		return nil, err
	}
	apiURL, err := url.ParseRequestURI(cfg.APIUrl)
	if err != nil || (apiURL.Scheme != "http" && apiURL.Scheme != "https") || apiURL.Host == "" {
		return nil, fmt.Errorf("APIUrl %q must be an absolute HTTP(S) URL", cfg.APIUrl)
	}

	authInterceptor := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+cfg.Token)
			return next(ctx, req)
		}
	})

	httpClient := &http.Client{Timeout: requestTimeout}
	return &Client{
		registry: registryv1beta1connect.NewRegistryServiceClient(httpClient, cfg.APIUrl, connect.WithInterceptors(authInterceptor)),
		projects: corev1connect.NewProjectServiceClient(httpClient, cfg.APIUrl, connect.WithInterceptors(authInterceptor)),
	}, nil
}

// ListImages returns one page (1-based) of tagged images in the repository.
func (c *Client) ListImages(ctx context.Context, repository string, page int32) ([]Image, bool, error) {
	//nolint:exhaustruct // TagQuery is an optional filter
	resp, err := c.registry.ListImages(ctx, connect.NewRequest(&registryv1beta1.ListImagesRequest{
		Repository: repository,
		Page:       page,
		PageSize:   listImagesPageSize,
		// Untagged manifests have nothing to reconcile against MySQL and
		// are reclaimed by Depot itself.
		TagStatus: registryv1beta1.ImageTagStatus_IMAGE_TAG_STATUS_TAGGED,
	}))
	if err != nil {
		return nil, false, fmt.Errorf("list images page %d of %q: %w", page, repository, err)
	}

	images := make([]Image, 0, len(resp.Msg.GetImages()))
	for _, img := range resp.Msg.GetImages() {
		pushedAt := img.GetPushedAt()
		if err := pushedAt.CheckValid(); err != nil {
			return nil, false, fmt.Errorf("image in repository %q has invalid pushed_at: %w", repository, err)
		}
		images = append(images, Image{
			Tags:     img.GetTags(),
			PushedAt: pushedAt.AsTime(),
		})
	}
	return images, resp.Msg.GetHasMore(), nil
}

// DeleteTag deletes one tag. An already-deleted tag is success so callers
// stay idempotent across retries.
func (c *Client) DeleteTag(ctx context.Context, repository, tag string) error {
	_, err := c.registry.DeleteTag(ctx, connect.NewRequest(&registryv1beta1.DeleteTagRequest{
		Repository: repository,
		Tag:        tag,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("delete tag %q in repository %q: %w", tag, repository, err)
	}
	return nil
}

// ListProjects returns one page of Depot projects. Pass the previous
// page's nextPageToken to continue; an empty return token means the last
// page.
func (c *Client) ListProjects(ctx context.Context, pageToken string) ([]Project, string, error) {
	//nolint:exhaustruct // RegionId/PageSize are optional filters
	req := &corev1.ListProjectsRequest{}
	if pageToken != "" {
		req.PageToken = &pageToken
	}
	resp, err := c.projects.ListProjects(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, "", fmt.Errorf("list projects: %w", err)
	}

	projects := make([]Project, 0, len(resp.Msg.GetProjects()))
	for _, p := range resp.Msg.GetProjects() {
		createdAt := p.GetCreatedAt()
		if err := createdAt.CheckValid(); err != nil {
			return nil, "", fmt.Errorf("depot project %q has invalid created_at: %w", p.GetProjectId(), err)
		}
		projects = append(projects, Project{
			ID:        p.GetProjectId(),
			Name:      p.GetName(),
			CreatedAt: createdAt.AsTime(),
		})
	}
	return projects, resp.Msg.GetNextPageToken(), nil
}

// DeleteProject deletes a Depot project (terminating its machines and
// build cache). An already-deleted project is success.
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	_, err := c.projects.DeleteProject(ctx, connect.NewRequest(&corev1.DeleteProjectRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return nil
		}
		return fmt.Errorf("delete depot project %q: %w", projectID, err)
	}
	return nil
}

// Noop is the API for environments without Depot: nothing was ever pushed
// or created there, so lists are empty and deletes succeed.
type Noop struct{}

var _ API = Noop{}

// NewNoop returns a Noop.
func NewNoop() Noop {
	return Noop{}
}

// ListImages returns an empty final page.
func (Noop) ListImages(context.Context, string, int32) ([]Image, bool, error) {
	return nil, false, nil
}

// DeleteTag treats the tag as already absent.
func (Noop) DeleteTag(context.Context, string, string) error {
	return nil
}

// ListProjects returns an empty final page.
func (Noop) ListProjects(context.Context, string) ([]Project, string, error) {
	return nil, "", nil
}

// DeleteProject treats the project as already absent.
func (Noop) DeleteProject(context.Context, string) error {
	return nil
}
