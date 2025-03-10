package membership

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/go/pkg/events"
)

// bus implements the memberlist.EventDelegate interface to handle and broadcast
// cluster membership events through typed channels.
type bus struct {
	onJoin   events.Topic[Member]
	onLeave  events.Topic[Member]
	onUpdate events.Topic[Member]
}

// NotifyJoin is called when a node joins the cluster.
// It broadcasts a join event with the new member's information to all subscribers.
// The Node argument must not be modified.
func (b *bus) NotifyJoin(node *memberlist.Node) {

	b.onJoin.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})
}

// NotifyLeave is called when a node leaves the cluster.
// It broadcasts a leave event with the departing member's information to all subscribers.
// The Node argument must not be modified.
func (b *bus) NotifyLeave(node *memberlist.Node) {
	b.onLeave.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})
}

// NotifyUpdate is called when a node's metadata is updated.
// It broadcasts an update event with the member's updated information to all subscribers.
// The Node argument must not be modified.
func (b *bus) NotifyUpdate(node *memberlist.Node) {

	b.onUpdate.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})

}
