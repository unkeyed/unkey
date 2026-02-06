---
title: sentinel
description: "provides the SentinelController for managing sentinel"
---

Package sentinel provides the SentinelController for managing sentinel Deployments and Services in Kubernetes.

The SentinelController is one half of krane's split control loop architecture. It operates independently from the DeploymentController, with its own:

  - Control plane sync stream (SyncSentinels)
  - Version cursor for resumable streaming
  - Circuit breaker for failure isolation
  - Kubernetes watch and refresh loops

### Architecture

The controller runs three loops for reliability:

  - \[Controller.runDesiredStateApplyLoop]: Receives desired state from the control plane's SyncSentinels stream and applies it to Kubernetes.

  - \[Controller.runActualStateReportLoop]: Watches Kubernetes for Deployment changes and reports actual state back to the control plane.

  - \[Controller.runResyncLoop]: Periodically re-queries the control plane for each existing sentinel to ensure eventual consistency.

### Resource Management

Sentinels are infrastructure proxies that route traffic to user deployments. Each sentinel gets both a Kubernetes Deployment (for the actual pods) and a ClusterIP Service (for stable in-cluster addressing). The Service is owned by the Deployment, so deleting the Deployment automatically cleans up the Service.

### Failure Isolation

By running as an independent controller, sentinel reconciliation continues even if deployment reconciliation is experiencing failures. Each controller has its own circuit breaker, so errors in one don't affect the other.

### Usage

	ctrl := sentinel.New(sentinel.Config{
	    ClientSet: kubeClient,
	    Cluster:   clusterClient,
	    Region:    "us-east-1",
	})

	if err := ctrl.Start(ctx); err != nil {
	    return fmt.Errorf("failed to start sentinel controller: %w", err)
	}
	defer ctrl.Stop()

## Constants

```go
const (
	// NamespaceSentinel is the Kubernetes namespace where sentinel pods run.
	NamespaceSentinel = "sentinel"

	// SentinelPort is the port sentinel pods listen on.
	SentinelPort = 8040

	// SentinelNodeClass is the node class for sentinel workloads.
	SentinelNodeClass = "sentinel"

	// fieldManagerKrane identifies krane as the server-side apply field manager.
	fieldManagerKrane = "krane"

	// topologyKeyZone is the standard Kubernetes topology key for availability zones
	topologyKeyZone = "topology.kubernetes.io/zone"
)
```


## Variables

sentinelToleration allows sentinel pods to be scheduled on sentinel nodes.
```go
var sentinelToleration = corev1.Toleration{
	Key:      "node-class",
	Operator: corev1.TolerationOpEqual,
	Value:    SentinelNodeClass,
	Effect:   corev1.TaintEffectNoSchedule,
}
```


## Functions


## Types

### type Config

```go
type Config struct {
	ClientSet kubernetes.Interface
	Cluster   ctrlv1connect.ClusterServiceClient
	Region    string
}
```

Config holds the configuration required to create a new \[Controller].

### type Controller

```go
type Controller struct {
	clientSet       kubernetes.Interface
	cluster         ctrlv1connect.ClusterServiceClient
	cb              circuitbreaker.CircuitBreaker[any]
	done            chan struct{}
	stopOnce        sync.Once
	region          string
	versionLastSeen uint64
}
```

Controller manages sentinel Deployments and Services in a Kubernetes cluster.

It maintains bidirectional state synchronization with the control plane: receiving desired state via WatchSentinels and reporting actual state via ReportSentinelStatus. The controller operates independently from the DeploymentController with its own version cursor and circuit breaker.

#### func New

```go
func New(cfg Config) *Controller
```

New creates a \[Controller] ready to be started with \[Controller.Start].

#### func (Controller) ApplySentinel

```go
func (c *Controller) ApplySentinel(ctx context.Context, req *ctrlv1.ApplySentinel) error
```

ApplySentinel creates or updates a sentinel as a Kubernetes Deployment with a Service.

Sentinels are infrastructure proxies that route traffic to user deployments within an environment. Each sentinel gets both a Deployment (for the actual pods) and a ClusterIP Service (for stable in-cluster addressing). The Service is owned by the Deployment, so deleting the Deployment automatically cleans up the Service.

ApplySentinel reports the available replica count back to the control plane after applying, so the platform knows when the sentinel is ready to receive traffic.

#### func (Controller) DeleteSentinel

```go
func (c *Controller) DeleteSentinel(ctx context.Context, req *ctrlv1.DeleteSentinel) error
```

DeleteSentinel removes a sentinel's Service and Deployment from the cluster.

Both resources are deleted explicitly rather than relying on owner reference cascading, ensuring cleanup completes even if ownership wasn't set correctly. Not-found errors are ignored since the desired end state is already achieved.

#### func (Controller) Start

```go
func (c *Controller) Start(ctx context.Context) error
```

Start launches the three background control loops:

  - \[Controller.runDesiredStateApplyLoop]: Receives desired state from the control plane's SyncSentinels stream and applies it to Kubernetes.

  - \[Controller.runActualStateReportLoop]: Watches Kubernetes for Deployment changes and reports actual state back to the control plane.

  - \[Controller.runResyncLoop]: Periodically re-queries the control plane for each existing sentinel to ensure eventual consistency.

All loops continue until the context is cancelled or \[Controller.Stop] is called.

#### func (Controller) Stop

```go
func (c *Controller) Stop() error
```

Stop signals all background goroutines to terminate.

