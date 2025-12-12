package sentinelcontroller

import (
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *SentinelController) BufferEvent(evt *ctrlv1.SentinelEvent) {
	c.events.Buffer(evt)
}
