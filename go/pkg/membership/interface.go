package membership

import "github.com/unkeyed/unkey/go/pkg/discovery"

// Membership defines the interface for cluster membership management.
// It provides functionality for node discovery, cluster joining/leaving,
// member listing, and membership event subscriptions.
//
// Implementations of this interface use a gossip protocol to maintain
// eventually consistent cluster state between all nodes.
//
// Typical usage:
//
//	membership, err := membership.New(config)
//	if err != nil {
//	    return err
//	}
//
//	// Discover and join other nodes
//	err = membership.Start(discoverer)
//
//	// Listen for membership events
//	joins := membership.SubscribeJoinEvents()
//	for node := range joins {
//	    // Handle new node
//	}
type Membership interface {
	// Start initializes the membership system and joins the cluster using
	// the provided discovery mechanism. It should be called only once.
	//
	// Start will return an error if the membership system is already running,
	// if node discovery fails, or if joining the cluster fails.
	Start(discovery.Discoverer) error

	// Leave gracefully removes the node from the cluster and shuts down
	// the membership system.
	//
	// It returns an error if the leave operation fails or times out.
	Leave() error

	// Members returns a list of all currently active members in the cluster.
	Members() ([]Member, error)

	// SubscribeJoinEvents returns a channel that receives Member events
	// whenever a new node joins the cluster.
	//
	// The returned channel will be closed when the membership system shuts down.
	SubscribeJoinEvents() <-chan Member

	// SubscribeUpdateEvents returns a channel that receives Member events
	// whenever a node changes its configuration in the cluster.
	//
	// The returned channel will be closed when the membership system shuts down.
	SubscribeUpdateEvents() <-chan Member

	// SubscribeLeaveEvents returns a channel that receives Member events
	// whenever a node leaves the cluster.
	//
	// The returned channel will be closed when the membership system shuts down.
	SubscribeLeaveEvents() <-chan Member

	// Self returns information about the local node for use in testing
	// and diagnostics.
	Self() Member
}
