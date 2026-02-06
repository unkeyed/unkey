package cluster

import (
	"sync"
	"sync/atomic"
	"time"
)

// NewNoop returns a no-op cluster that does not participate in gossip.
// All operations are safe to call but do nothing.
func NewNoop() *Cluster {
	return &Cluster{
		config: Config{
			Region:      "",
			NodeID:      "",
			BindAddr:    "",
			BindPort:    0,
			WANBindPort: 0,
			LANSeeds:    nil,
			WANSeeds:    nil,
			OnMessage:   nil,
		},
		mu:        sync.RWMutex{},
		lan:       nil,
		lanQueue:  nil,
		wan:       nil,
		wanQueue:  nil,
		isGateway: false,
		joinTime:  time.Time{},
		noop:      true,
		closing:   atomic.Bool{},
		evalCh:    nil,
		done:      nil,
	}
}
