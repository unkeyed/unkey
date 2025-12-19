package sentinelreflector

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
)

func (r *Reflector) updateState(ctx context.Context, req *ctrlv1.UpdateSentinelStateRequest) error {

	_, err := r.cb.Do(ctx, func(innerCtx context.Context) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {
		return r.cluster.UpdateSentinelState(innerCtx, connect.NewRequest(req))
	})

	return err
}
