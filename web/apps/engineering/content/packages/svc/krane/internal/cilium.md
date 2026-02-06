---
title: cilium
description: "manages CiliumNetworkPolicy resources in Kubernetes clusters"
---

Package cilium manages CiliumNetworkPolicy resources in Kubernetes clusters.

The package provides a controller that synchronizes Cilium network policies between the control plane and Kubernetes clusters. It enables namespace-level network isolation by applying policies that control traffic between sentinel pods and customer deployment pods.

### Architecture

The \[Controller] runs two independent control loops:

  - \[Controller.runDesiredStateApplyLoop]: Streams policy updates from the control plane via WatchCiliumNetworkPolicies and applies them to Kubernetes.

  - \[Controller.runResyncLoop]: Periodically reconciles all existing policies against the control plane to handle missed events or drift.

### Usage

Create a controller with \[New] and start it with \[Controller.Start]:

	controller := cilium.New(cilium.Config{
	    ClientSet:     clientSet,
	    DynamicClient: dynamicClient,
	    Cluster:       clusterClient,
	    Region:        "us-east-1",
	})
	if err := controller.Start(ctx); err != nil {
	    return err
	}
	defer controller.Stop()

The controller uses server-side apply for CiliumNetworkPolicy resources, allowing concurrent modifications from different sources without conflicts.

## Types

### type Config

```go
type Config struct {
	// ClientSet provides typed Kubernetes API access for Kubernetes operations.
	ClientSet kubernetes.Interface

	// DynamicClient provides unstructured Kubernetes API access for CiliumNetworkPolicy
	// resources that don't have generated Go types.
	DynamicClient dynamic.Interface

	// Cluster is the control plane RPC client for WatchCiliumNetworkPolicies calls.
	Cluster ctrlv1connect.ClusterServiceClient

	// Region identifies the cluster region for filtering policy streams.
	Region string
}
```

Config holds the configuration required to create a new \[Controller].

All fields are required. The ClientSet and DynamicClient are used for Kubernetes operations, while Cluster provides the control plane RPC client for state synchronization. Region determines which policies this controller manages.

### type Controller

```go
type Controller struct {
	clientSet       kubernetes.Interface
	dynamicClient   dynamic.Interface
	cluster         ctrlv1connect.ClusterServiceClient
	done            chan struct{}
	region          string
	versionLastSeen uint64
}
```

Controller manages CiliumNetworkPolicy resources in a Kubernetes cluster by maintaining bidirectional state synchronization with the control plane.

The controller receives desired state via the WatchCiliumNetworkPolicies stream. It operates independently from the sentinel controller with its own version cursor and circuit breaker, ensuring that failures in one controller don't cascade to the other.

Create a Controller with \[New] and start it with \[Controller.Start]. The controller runs until the context is cancelled or \[Controller.Stop] is called.

#### func New

```go
func New(cfg Config) *Controller
```

New creates a \[Controller] ready to be started with \[Controller.Start].

The controller initializes with versionLastSeen=0, meaning it will receive all pending policies on first connection. The circuit breaker starts in a closed (healthy) state.

#### func (Controller) ApplyCiliumNetworkPolicy

```go
func (c *Controller) ApplyCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.ApplyCiliumNetworkPolicy) error
```

ApplyCiliumNetworkPolicy creates or updates a CiliumNetworkPolicy in the cluster.

The method uses server-side apply with the dynamic client, enabling concurrent modifications from different sources without conflicts.

ApplyCiliumNetworkPolicy validates required fields and returns an error if any are missing or invalid: K8sNamespace, K8sName, CiliumNetworkPolicyId, and Policy must be non-empty.

#### func (Controller) DeleteCiliumNetworkPolicy

```go
func (c *Controller) DeleteCiliumNetworkPolicy(ctx context.Context, req *ctrlv1.DeleteCiliumNetworkPolicy) error
```

DeleteCiliumNetworkPolicy removes a CiliumNetworkPolicy from the cluster.

Not-found errors are ignored since the desired end state (resource gone) is already achieved. The method is idempotent: calling it multiple times for the same policy succeeds without error.

#### func (Controller) Start

```go
func (c *Controller) Start(ctx context.Context) error
```

Start launches the background control loops and blocks until they're initialized.

The method starts \[Controller.runResyncLoop] and \[Controller.runDesiredStateApplyLoop] as background goroutines.

All loops continue until the context is cancelled or \[Controller.Stop] is called.

#### func (Controller) Stop

```go
func (c *Controller) Stop() error
```

Stop signals all background goroutines to terminate by closing the done channel. Returns nil; the error return exists for interface compatibility.

