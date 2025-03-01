package membership

import "github.com/unkeyed/unkey/go/pkg/discovery"

// Membership defines the interface for cluster membership management.
// It provides functionality for node discovery, cluster joining/leaving,
// member listing, and membership event subscriptions. The implementation
// uses a gossip protocol to maintain eventually consistent cluster state.
type Membership interface {
	// Start initializes the membership system and joins the cluster using
	// the provided discovery mechanism. It should only be called once.
	//
	// Parameters:
	//   - discovery: A Discoverer implementation to find other cluster nodes
	//
	// Returns an error if:
	//   - The membership system is already started
	//   - Node discovery fails
	//   - Joining the cluster fails
	Start(discovery.Discoverer) error

	// Leave gracefully removes the node from the cluster and shuts down
	// the membership system.
	//
	// Returns an error if the leave operation fails or times out.
	Leave() error

	// Members returns a list of all currently active members in the cluster.
	//
	// Returns:
	//   - []Member: Slice of active cluster members
	//   - error: If there was an error retrieving the member list
	Members() ([]Member, error)

	// SubscribeJoinEvents returns a channel that receives Member events
	// whenever a new node joins the cluster.
	//
	// The returned channel will be closed when the membership system is shut down.
	SubscribeJoinEvents() <-chan Member

	// SubscribeLeaveEvents returns a channel that receives Member events
	// whenever a node leaves the cluster.
	//
	// The returned channel will be closed when the membership system is shut down.
	SubscribeLeaveEvents() <-chan Member

	// Self returns information about the local node
	// for use in tests and diagnostics.
	Self() Member
}
