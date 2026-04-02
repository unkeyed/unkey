// Package balancer provides load balancing strategies for selecting instances
// to route requests to.
//
// The core abstraction is the [Balancer] interface, which requires only a
// [Balancer.Pick] method. Balancers that need to track in-flight requests
// (such as [P2CBalancer]) additionally implement [InflightTracker].
//
// Usage:
//
//	b := balancer.NewP2CBalancer()
//	idx := b.Pick(instanceIDs)
//	b.Acquire(instanceIDs[idx])
//	defer b.Release(instanceIDs[idx])
package balancer
