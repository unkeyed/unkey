package privatenetwork

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (r *Reconciler) snapshot(ctx context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
	stream, err := r.cluster.StreamPrivateNetworkState(ctx, &ctrlv1.StreamPrivateNetworkStateRequest{Cluster: r.clusterKey})
	if err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("close private network snapshot stream", "error", err)
		}
	}()

	var apps []*ctrlv1.PrivateNetworkApp
	for stream.Receive() {
		chunk := stream.Msg()
		if chunk.GetComplete() {
			if chunk.GetTotal() != uint64(len(apps)) || len(chunk.GetApps()) != 0 {
				return nil, fmt.Errorf("private network snapshot has %d apps, complete chunk reports %d", len(apps), chunk.GetTotal())
			}
			return apps, nil
		}
		apps = append(apps, chunk.GetApps()...)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	return nil, fmt.Errorf("private network snapshot ended before it was complete")
}

type snapshotRejection struct {
	retainedBindingKey string
	err                error
}

func validateSnapshot(apps []*ctrlv1.PrivateNetworkApp) ([]*ctrlv1.PrivateNetworkApp, []snapshotRejection) {
	identities := make(map[string]int, len(apps))
	for _, app := range apps {
		if app != nil {
			identities[appIdentity(app)]++
		}
	}

	eligible := make([]*ctrlv1.PrivateNetworkApp, 0, len(apps))
	var rejected []snapshotRejection
	for i, app := range apps {
		if app == nil {
			rejected = append(rejected, snapshotRejection{retainedBindingKey: "", err: fmt.Errorf("invalid private network snapshot app %d: app is nil", i)})
			continue
		}
		err := assert.All(
			assert.NotEmpty(app.GetWorkspaceId(), "workspace ID is required"),
			assert.NotEmpty(app.GetProjectId(), "project ID is required"),
			assert.NotEmpty(app.GetAppId(), "app ID is required"),
			assert.NotEmpty(app.GetCallerDeploymentId(), "caller deployment ID is required"),
			assert.NotEmpty(app.GetBindingId(), "binding ID is required"),
			assert.Equal(len(validation.IsValidLabelValue(app.GetCallerDeploymentId())), 0, "invalid caller deployment ID"),
			assert.Equal(len(validation.IsValidLabelValue(app.GetBindingId())), 0, "invalid binding ID"),
			assert.Equal(len(validation.IsDNS1123Label(app.GetBindingName())), 0, "invalid binding name"),
			assert.Equal(len(validation.IsDNS1123Label(app.GetK8SNamespace())), 0, "invalid Kubernetes namespace"),
			assert.True(app.GetDeploymentId() == "" || app.GetPort() > 0, "resolved target port must be positive"),
			assert.GreaterOrEqual(app.GetPort(), int32(0), "port must not be negative"),
			assert.LessOrEqual(app.GetPort(), int32(65535), "port must be at most 65535"),
			assert.Equal(identities[appIdentity(app)], 1, "duplicate app identity"),
		)
		if err != nil {
			rejected = append(rejected, snapshotRejection{
				retainedBindingKey: publishedBindingKey(app),
				err:                fmt.Errorf("invalid private network snapshot app %d: %w", i, err),
			})
			continue
		}
		eligible = append(eligible, app)
	}
	return eligible, rejected
}

func appIdentity(app *ctrlv1.PrivateNetworkApp) string {
	return app.GetWorkspaceId() + "/" + app.GetProjectId() + "/" + app.GetCallerDeploymentId() + "/" + app.GetBindingName()
}

func publishedBindingKey(app *ctrlv1.PrivateNetworkApp) string {
	if len(validation.IsDNS1123Label(app.GetK8SNamespace())) != 0 || app.GetBindingId() == "" || app.GetCallerDeploymentId() == "" {
		return ""
	}
	return app.GetK8SNamespace() + "/" + bindingResourceName(app)
}
