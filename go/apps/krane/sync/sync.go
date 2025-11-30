package sync

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
)

func (s *SyncEngine) pull() error {

	s.logger.Info("starting pull from control plane")
	httpClient := &http.Client{}

	// TODO: add auth
	ctrlClient := ctrlv1connect.NewClusterServiceClient(httpClient, s.controlPlaneUrl)

	stream, err := ctrlClient.GetDesiredState(context.Background(), connect.NewRequest(&ctrlv1.GetDesiredStateRequest{
		ClientId: s.instanceID,
		Selectors: map[string]string{
			"region": s.region,
		},
	}))
	if err != nil {
		return err
	}

	totalEvents := 0
	for stream.Receive() {

		s.logger.Info("received desired state")
		s.events.Buffer(stream.Msg())
	}
	err = stream.Err()
	if err != nil {
		return err
	}
	s.logger.Info("completed pull from control plane",
		"total_events", totalEvents,
	)

	return nil

}
