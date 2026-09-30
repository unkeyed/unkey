package privatenetwork

import (
	"context"
	"fmt"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"k8s.io/apimachinery/pkg/util/validation"
)

func (r *Reconciler) snapshot(ctx context.Context) ([]*ctrlv1.PrivateNetworkBinding, error) {
	stream, err := r.cluster.StreamPrivateNetworkState(ctx, &ctrlv1.StreamPrivateNetworkStateRequest{Cluster: r.clusterKey})
	if err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	defer func() {
		if err := stream.Close(); err != nil {
			logger.Warn("close private network snapshot stream", "error", err)
		}
	}()

	var snapshotBindings []*ctrlv1.PrivateNetworkBinding
	for stream.Receive() {
		chunk := stream.Msg()
		if chunk.GetComplete() {
			if chunk.GetTotal() != uint64(len(snapshotBindings)) || len(chunk.GetBindings()) != 0 {
				return nil, fmt.Errorf("private network snapshot has %d bindings, complete chunk reports %d", len(snapshotBindings), chunk.GetTotal())
			}
			return snapshotBindings, nil
		}
		snapshotBindings = append(snapshotBindings, chunk.GetBindings()...)
	}
	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("get complete private network snapshot: %w", err)
	}
	return nil, fmt.Errorf("private network snapshot ended before it was complete")
}

type snapshotRejection struct {
	bindingSpec        *ctrlv1.PrivateNetworkBinding
	retainedBindingKey string
	err                error
}

func validateSnapshot(snapshotBindings []*ctrlv1.PrivateNetworkBinding) ([]*ctrlv1.PrivateNetworkBinding, []snapshotRejection) {
	identities := make(map[string]int, len(snapshotBindings))
	for _, bindingSpec := range snapshotBindings {
		if bindingSpec != nil {
			identities[bindingIdentity(bindingSpec)]++
		}
	}

	eligible := make([]*ctrlv1.PrivateNetworkBinding, 0, len(snapshotBindings))
	var rejected []snapshotRejection
	for i, bindingSpec := range snapshotBindings {
		if bindingSpec == nil {
			rejected = append(rejected, snapshotRejection{bindingSpec: nil, retainedBindingKey: "", err: fmt.Errorf("invalid private network snapshot binding %d: binding is nil", i)})
			continue
		}
		err := assert.All(
			assert.NotEmpty(bindingSpec.GetWorkspaceId(), "workspace ID is required"),
			assert.NotEmpty(bindingSpec.GetProjectId(), "project ID is required"),
			assert.NotEmpty(bindingSpec.GetTargetAppId(), "target app ID is required"),
			assert.NotEmpty(bindingSpec.GetCallerDeploymentId(), "caller deployment ID is required"),
			assert.NotEmpty(bindingSpec.GetBindingId(), "binding ID is required"),
			assert.Equal(len(validation.IsValidLabelValue(bindingSpec.GetCallerDeploymentId())), 0, "invalid caller deployment ID"),
			assert.Equal(len(validation.IsValidLabelValue(bindingSpec.GetBindingId())), 0, "invalid binding ID"),
			assert.Equal(len(validation.IsDNS1123Label(bindingSpec.GetBindingName())), 0, "invalid binding name"),
			assert.Equal(len(validation.IsDNS1123Label(bindingSpec.GetK8SNamespace())), 0, "invalid Kubernetes namespace"),
			assert.True(bindingSpec.GetTargetDeploymentId() == "" || bindingSpec.GetTargetPort() > 0, "resolved target port must be positive"),
			assert.GreaterOrEqual(bindingSpec.GetTargetPort(), int32(0), "port must not be negative"),
			assert.LessOrEqual(bindingSpec.GetTargetPort(), int32(65535), "port must be at most 65535"),
			assert.Equal(identities[bindingIdentity(bindingSpec)], 1, "duplicate binding identity"),
		)
		if err != nil {
			rejected = append(rejected, snapshotRejection{
				bindingSpec:        bindingSpec,
				retainedBindingKey: publishedBindingKey(bindingSpec),
				err:                fmt.Errorf("invalid private network snapshot binding %d: %w", i, err),
			})
			continue
		}
		eligible = append(eligible, bindingSpec)
	}
	return eligible, rejected
}

func bindingIdentity(bindingSpec *ctrlv1.PrivateNetworkBinding) string {
	return bindingSpec.GetWorkspaceId() + "/" + bindingSpec.GetProjectId() + "/" + bindingSpec.GetCallerDeploymentId() + "/" + bindingSpec.GetBindingName()
}

func publishedBindingKey(bindingSpec *ctrlv1.PrivateNetworkBinding) string {
	if len(validation.IsDNS1123Label(bindingSpec.GetK8SNamespace())) != 0 || bindingSpec.GetBindingId() == "" || bindingSpec.GetCallerDeploymentId() == "" {
		return ""
	}
	return bindingSpec.GetK8SNamespace() + "/" + bindingResourceName(bindingSpec)
}
