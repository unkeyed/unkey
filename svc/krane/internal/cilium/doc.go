// Package cilium manages CiliumNetworkPolicy resources in Kubernetes clusters.
//
// The package provides a controller that synchronizes Cilium network policies between
// the control plane and Kubernetes clusters. It enables namespace-level network
// isolation by applying policies that control traffic between sentinel pods and
// customer deployment pods.
//
// # Architecture
//
// The [Controller] runs two independent control loops:
//
//   - [Controller.runDesiredStateApplyLoop]: Streams policy updates from the control
//     plane via WatchCiliumNetworkPolicies and applies them to Kubernetes.
//
//   - [Controller.runResyncLoop]: Periodically reconciles all existing policies against
//     the control plane to handle missed events or drift.
//
// # Usage
//
// Create a controller with [New] and start it with [Controller.Start]:
//
//	controller := cilium.New(cilium.Config{
//	    ClientSet:     clientSet,
//	    DynamicClient: dynamicClient,
//	    Cluster:       clusterClient,
//	    Region:        "us-east-1",
//	})
//	if err := controller.Start(ctx); err != nil {
//	    return err
//	}
//	defer controller.Stop()
//
// The controller uses server-side apply for CiliumNetworkPolicy resources, allowing
// concurrent modifications from different sources without conflicts.
package cilium
