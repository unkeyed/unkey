package cdc

import (
	"context"

	"vitess.io/vitess/go/vt/proto/vtgate"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

// streamResult carries a response or error from the read goroutine.
type streamResult struct {
	response *vtgate.VStreamResponse
	err      error
}

// readVStream lets Watch time out a blocked Recv call.
// It reads at most one response ahead and stops when the context is canceled.
func readVStream(ctx context.Context, stream vtgateservice.Vitess_VStreamClient) <-chan streamResult {
	responses := make(chan streamResult)
	go func() {
		for {
			response, err := stream.Recv()
			select {
			case responses <- streamResult{response: response, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return responses
}
