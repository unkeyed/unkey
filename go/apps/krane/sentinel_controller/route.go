package sentinelcontroller

import (
	"context"

	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (c *SentinelController) route() {

	for e := range c.events.Consume() {
		ctx := context.Background()
		switch x := e.GetEvent().(type) {
		case *ctrlv1.SentinelEvent_Apply:
			if err := c.ApplySentinel(ctx, x.Apply); err != nil {
				c.logger.Error("unable to apply sentinel", "sentinel_id", x.Apply.GetSentinelId(), "error", err.Error())
			}
		case *ctrlv1.SentinelEvent_Delete:
			if err := c.DeleteSentinel(ctx, x.Delete); err != nil {
				c.logger.Error("unable to delete sentinel", "sentinel_id", x.Delete.GetSentinelId(), "error", err.Error())
			}
		}

	}
}
