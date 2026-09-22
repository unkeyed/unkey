package deployment

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestReconcileDesiredState_NotFoundRejectsLaterStaleApply(t *testing.T) {
	replicaSet := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name:      "deployment",
		Namespace: "workspace",
		Labels:    labels.New().DeploymentID("dep_1"),
	}}
	client := fake.NewSimpleClientset(replicaSet)
	reportErr := errors.New("report failed")
	cluster := &testutil.MockClusterClient{
		GetDesiredDeploymentStateFunc: func(context.Context, *ctrlv1.GetDesiredDeploymentStateRequest) (*ctrlv1.DeploymentState, error) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("topology not found"))
		},
		ReportDeploymentStatusFunc: func(context.Context, *ctrlv1.ReportDeploymentStatusRequest) (*ctrlv1.ReportDeploymentStatusResponse, error) {
			return nil, reportErr
		},
	}
	controller := New(Config{
		ClientSet:    client,
		Cluster:      cluster,
		Fingerprints: cache.NewNoopCache[string, string](),
	})

	controller.reconcileDesiredState(t.Context(), replicaSet)
	staleApplied := false
	require.NoError(t, controller.reconcileRevision("dep_1", 100, func() error {
		staleApplied = true
		return nil
	}))

	require.False(t, staleApplied)
}
