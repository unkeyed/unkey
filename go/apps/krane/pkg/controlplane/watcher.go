package controlplane

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

// Watcher provides event streaming capabilities from the control plane service.
//
// The watcher handles both live streaming for real-time events and periodic
// synchronization for full state reconciliation. It implements automatic
// reconnection with exponential backoff and provides a buffered channel
// for consuming events.
//
// The type parameter T represents the specific event type being watched.
type Watcher struct {
	logger    logging.Logger
	clusterID string
	region    string
	cluster   ctrlv1connect.ClusterServiceClient
}

// WatcherConfig holds the configuration for creating a new Watcher.
//
// All fields are required for proper watcher operation. The CreateStream
// function should typically be a method on a control plane client that
// establishes the streaming connection.
type WatcherConfig struct {
	// Logger is used for logging watcher events and errors.
	Logger logging.Logger

	// Cluster	ID uniquely identifies this watcher instance to the control plane.
	ClusterID string

	// Region identifies the geographical region for filtering events.
	Region string

	// Cluster is the control plane client used to establish the streaming connection.
	Cluster ctrlv1connect.ClusterServiceClient
}

// NewWatcher creates a new event watcher with the specified configuration.
//
// The watcher is initialized with a 1000-event buffer that does not drop
// events when full. Events are buffered internally and made available through
// the Consume() method.
//
// The returned watcher is safe for concurrent use, but Watch() and Sync()
// should typically be run in separate goroutines.
func NewWatcher(cfg WatcherConfig) *Watcher {
	w := &Watcher{
		logger:    cfg.Logger,
		clusterID: cfg.ClusterID,
		region:    cfg.Region,
		cluster:   cfg.Cluster,
	}

	return w
}

func (w *Watcher) Start(ctx context.Context, handle func(context.Context, *ctrlv1.State) error) {

	repeat.Every(time.Minute, func() {
		w.logger.Info("Pulling synthetic state from control plane")
		stream, err := w.cluster.Watch(ctx, connect.NewRequest(&ctrlv1.WatchRequest{
			ClusterId: w.clusterID,
			Region:    w.region,
			Synthetic: true,
			Live:      false,
		}))
		if err != nil {
			w.logger.Error("unable to connect to control plane", "error", err)
			return
		}
		for stream.Receive() {
			if err := handle(ctx, stream.Msg()); err != nil {
				w.logger.Error("error handling state", "error", err)
			}
		}
		err = stream.Close()
		if err != nil {
			w.logger.Error("unable to close stream", "error", err)
		}

	})

	go func() {
		w.logger.Info("Starting control plane watcher")

		consecutiveFailures := 0
		var stream *connect.ServerStreamForClient[ctrlv1.State]
		var err error
		for {
			if stream == nil {
				stream, err = w.cluster.Watch(ctx, connect.NewRequest(&ctrlv1.WatchRequest{
					ClusterId: w.clusterID,
					Region:    w.region,
					Synthetic: false,
					Live:      true,
				}))
				if err != nil {
					consecutiveFailures++
					w.logger.Error("unable to connect to control plane", "consecutive_failures", consecutiveFailures)
					time.Sleep(time.Duration(min(60, consecutiveFailures)) * time.Second)
					continue
				} else {
					consecutiveFailures = 0
				}
			}
			w.logger.Info("control plane watch stream started")

			hasMsg := stream.Receive()
			if !hasMsg {
				w.logger.Info("Stream ended, reconnecting...",
					"error", stream.Err(),
				)
				stream = nil
				time.Sleep(time.Second)
				continue
			}
			msg := stream.Msg()
			w.logger.Info("control plane watch stream received message", "message", msg)
			if err := handle(ctx, msg); err != nil {
				w.logger.Error("error handling state", "error", err)
			}
		}
	}()

}
