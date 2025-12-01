package docker

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/krane/backend"
)

// DeleteGateway removes all containers for a gateway.
//
// Finds containers by gateway ID label and forcibly removes them with
// volumes and network links to ensure complete cleanup. This method is idempotent
// and will not fail if the containers are already deleted.
func (d *docker) DeleteGateway(ctx context.Context, req backend.DeleteGatewayRequest) error {
	d.logger.Info("deleting gateway", "gateway_id", req.GatewayID)

	return d.deleteByLabels(ctx, map[string]string{
		"unkey.gateway.id": req.GatewayID,
	})
}
