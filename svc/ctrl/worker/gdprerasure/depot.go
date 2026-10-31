package gdprerasure

import (
	"context"
	"errors"
	"net/http"
	"time"

	"buf.build/gen/go/depot/api/connectrpc/go/depot/core/v1/corev1connect"
	corev1 "buf.build/gen/go/depot/api/protocolbuffers/go/depot/core/v1"
	"connectrpc.com/connect"
)

// DepotProjects deletes the Depot projects the deploy workflow creates per
// Unkey project. They hold the customer's build context and layer cache.
type DepotProjects interface {
	// DeleteProject deletes the project and reports whether it existed. A
	// project that is already gone counts as deleted.
	DeleteProject(ctx context.Context, projectID string) (bool, error)
}

type depotProjects struct {
	client corev1connect.ProjectServiceClient
}

var _ DepotProjects = (*depotProjects)(nil)

// NewDepot builds a Depot-backed DepotProjects from the API URL and token the
// deploy workflow creates projects with.
func NewDepot(apiURL, token string) DepotProjects {
	auth := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
	return &depotProjects{client: corev1connect.NewProjectServiceClient(
		&http.Client{Timeout: 30 * time.Second},
		apiURL,
		connect.WithInterceptors(auth),
	)}
}

func (d *depotProjects) DeleteProject(ctx context.Context, projectID string) (bool, error) {
	_, err := d.client.DeleteProject(ctx, connect.NewRequest(&corev1.DeleteProjectRequest{ProjectId: projectID})) //nolint:exhaustruct // proto internals
	if cErr, ok := errors.AsType[*connect.Error](err); ok && cErr.Code() == connect.CodeNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
