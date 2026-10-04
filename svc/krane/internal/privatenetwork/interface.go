package privatenetwork

import (
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/clock"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Config holds the dependencies of a [Reconciler].
type Config struct {
	Client     kubernetes.Interface
	Dynamic    dynamic.Interface
	Cluster    ctrl.ClusterServiceClient
	ClusterKey *ctrlv1.ClusterKey
	Clock      clock.Clock
}
