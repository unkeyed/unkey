---
title: deployment
description: "manages user workload ReplicaSets in Kubernetes as part of"
---

Package deployment manages user workload ReplicaSets in Kubernetes as part of krane's split control loop architecture.

The package provides \[Controller], which operates independently from the sentinel controller with its own control plane stream, version cursor, and circuit breaker. This separation ensures deployment reconciliation continues even when sentinel reconciliation experiences failures.

### Architecture

The controller runs three concurrent loops for reliability:

\[Controller.runDesiredStateApplyLoop] streams desired state from the control plane's WatchDeployments RPC and applies changes to Kubernetes. It uses a version cursor for resumable streaming and automatically reconnects with jittered backoff on errors.

\[Controller.runActualStateReportLoop] watches Kubernetes for ReplicaSet changes and reports actual state back to the control plane via ReportDeploymentStatus. This keeps the control plane's routing tables synchronized with what's actually running.

\[Controller.runResyncLoop] runs every minute as a consistency safety net. While the other loops handle real-time events, they can miss updates during network partitions, controller restarts, or buffer overflows. The resync loop queries the control plane for each existing ReplicaSet and applies any drift.

### Security

All user workloads run with gVisor isolation (RuntimeClass "gvisor") since they execute untrusted code. Each namespace gets a CiliumNetworkPolicy that restricts ingress to only sentinels with matching workspace and environment IDs.

### Scheduling

Deployment pods are spread across availability zones using TopologySpreadConstraints with maxSkew=1 for high availability. Pod affinity prefers scheduling in the same zone as the environment's sentinels to minimize cross-AZ latency.

### Usage

	ctrl := deployment.New(deployment.Config{
	    ClientSet:     kubeClient,
	    DynamicClient: dynamicClient,
	    Cluster:       clusterClient,
	    Region:        "us-east-1",
	})

	if err := ctrl.Start(ctx); err != nil {
	    return fmt.Errorf("start deployment controller: %w", err)
	}
	defer ctrl.Stop()

## Constants

```go
const (
	// DeploymentPort is the port all user deployment containers expose. The routing
	// layer and sentinel proxies use this port to forward traffic to user code.
	DeploymentPort = 8080

	// runtimeClassGvisor specifies the gVisor sandbox RuntimeClass for running
	// untrusted user workloads with kernel-level isolation.
	runtimeClassGvisor = "gvisor"

	// fieldManagerKrane identifies krane as the server-side apply field manager,
	// so field ownership/conflict detection is tracked per manager.
	fieldManagerKrane = "krane"

	// CustomerNodeClass is the Karpenter nodepool name for untrusted customer
	// workloads. Nodes in this pool have additional isolation and monitoring.
	CustomerNodeClass = "untrusted"
)
```

```go
const (
	// topologyKeyZone is the standard Kubernetes label for availability zones,
	// used for spreading pods across zones for high availability.
	topologyKeyZone = "topology.kubernetes.io/zone"
)
```


## Variables

untrustedToleration allows deployment pods to be scheduled on nodes tainted for untrusted workloads. Without this toleration, pods would be rejected by the Karpenter-managed nodepool's NoSchedule taint.
```go
var untrustedToleration = corev1.Toleration{
	Key:      "karpenter.sh/nodepool",
	Operator: corev1.TolerationOpEqual,
	Value:    CustomerNodeClass,
	Effect:   corev1.TaintEffectNoSchedule,
}
```


## Functions


## Types

### type Config

```go
type Config struct {
	// ClientSet provides typed Kubernetes API access for ReplicaSet and Pod operations.
	ClientSet kubernetes.Interface

	// DynamicClient provides unstructured Kubernetes API access for CiliumNetworkPolicy
	// resources that don't have generated Go types.
	DynamicClient dynamic.Interface

	// Cluster is the control plane RPC client for WatchDeployments and
	// ReportDeploymentStatus calls.
	Cluster ctrlv1connect.ClusterServiceClient

	// Region identifies the cluster region for filtering deployment streams.
	Region string
}
```

Config holds the configuration required to create a new \[Controller].

All fields are required. The ClientSet and DynamicClient are used for Kubernetes operations, while Cluster provides the control plane RPC client for state synchronization. Region determines which deployments this controller manages.

### type Controller

```go
type Controller struct {
	clientSet       kubernetes.Interface
	dynamicClient   dynamic.Interface
	cluster         ctrlv1connect.ClusterServiceClient
	cb              circuitbreaker.CircuitBreaker[any]
	done            chan struct{}
	region          string
	versionLastSeen uint64
}
```

Controller manages deployment ReplicaSets in a Kubernetes cluster by maintaining bidirectional state synchronization with the control plane.

The controller receives desired state via the WatchDeployments stream and reports actual state via ReportDeploymentStatus. It operates independently from the sentinel controller with its own version cursor and circuit breaker, ensuring that failures in one controller don't cascade to the other.

Create a Controller with \[New] and start it with \[Controller.Start]. The controller runs until the context is cancelled or \[Controller.Stop] is called.

#### func New

```go
func New(cfg Config) *Controller
```

New creates a \[Controller] ready to be started with \[Controller.Start].

The controller initializes with versionLastSeen=0, meaning it will receive all pending deployments on first connection. The circuit breaker starts in a closed (healthy) state.

#### func (Controller) ApplyDeployment

```go
func (c *Controller) ApplyDeployment(ctx context.Context, req *ctrlv1.ApplyDeployment) error
```

ApplyDeployment creates or updates a user workload as a Kubernetes ReplicaSet.

The method uses server-side apply to create or update the ReplicaSet, enabling concurrent modifications from different sources without conflicts. After applying, it queries the resulting pods and reports their addresses and status to the control plane so the routing layer knows where to send traffic.

ApplyDeployment validates all required fields and returns an error if any are missing or invalid: WorkspaceId, ProjectId, EnvironmentId, DeploymentId, K8sNamespace, K8sName, and Image must be non-empty; Replicas must be >= 0; CpuMillicores and MemoryMib must be > 0.

The namespace is created automatically if it doesn't exist, along with a CiliumNetworkPolicy restricting ingress to matching sentinels. Pods run with gVisor isolation (RuntimeClass "gvisor") since they execute untrusted user code, and are scheduled on Karpenter-managed untrusted nodes with zone-spread constraints.

#### func (Controller) DeleteDeployment

```go
func (c *Controller) DeleteDeployment(ctx context.Context, req *ctrlv1.DeleteDeployment) error
```

DeleteDeployment removes a user workload's ReplicaSet from the cluster.

Not-found errors are ignored since the desired end state (resource gone) is already achieved. After deletion, the method reports the deletion to the control plane so it can update routing tables and stop sending traffic to this deployment.

The method is idempotent: calling it multiple times for the same deployment succeeds without error.

#### func (Controller) Start

```go
func (c *Controller) Start(ctx context.Context) error
```

Start launches the three background control loops and blocks until they're initialized.

The method starts \[Controller.runResyncLoop] and \[Controller.runDesiredStateApplyLoop] as background goroutines, and initializes \[Controller.runActualStateReportLoop]'s Kubernetes watch before returning. If watch initialization fails, Start returns the error and no goroutines are left running.

All loops continue until the context is cancelled or \[Controller.Stop] is called.

#### func (Controller) Stop

```go
func (c *Controller) Stop() error
```

Stop signals all background goroutines to terminate by closing the done channel. Returns nil; the error return exists for interface compatibility.

