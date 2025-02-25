package membership

import (
	"context"

	"github.com/hashicorp/memberlist"
	"github.com/unkeyed/unkey/go/pkg/events"
)

type bus struct {
	onJoin   events.Topic[Member]
	onLeave  events.Topic[Member]
	onUpdate events.Topic[Member]
}

// NotifyJoin is invoked when a node is detected to have joined.
// The Node argument must not be modified.
func (b *bus) NotifyJoin(node *memberlist.Node) {

	b.onJoin.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})
}

// NotifyLeave is invoked when a node is detected to have left.
// The Node argument must not be modified.
func (b *bus) NotifyLeave(node *memberlist.Node) {

	b.onLeave.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})
}

// NotifyUpdate is invoked when a node is detected to have
// updated, usually involving the meta data. The Node argument
// must not be modified.
func (b *bus) NotifyUpdate(node *memberlist.Node) {

	b.onUpdate.Emit(context.Background(), Member{
		NodeID: node.Name,
		Addr:   node.Addr.String(),
	})

}
