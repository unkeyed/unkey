package controlplane

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/buffer"
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
type Watcher[T any] struct {
	logger       logging.Logger
	instanceID   string
	region       string
	createStream func(context.Context, *connect.Request[ctrlv1.WatchRequest]) (*connect.ServerStreamForClient[T], error)
}

// WatcherConfig holds the configuration for creating a new Watcher.
//
// All fields are required for proper watcher operation. The CreateStream
// function should typically be a method on a control plane client that
// establishes the streaming connection.
type WatcherConfig[T any] struct {
	// Logger is used for logging watcher events and errors.
	Logger logging.Logger

	// InstanceID uniquely identifies this watcher instance to the control plane.
	InstanceID string

	// Region identifies the geographical region for filtering events.
	Region string

	// CreateStream is a function that establishes a streaming connection to the control plane.
	CreateStream func(context.Context, *connect.Request[ctrlv1.WatchRequest]) (*connect.ServerStreamForClient[T], error)
}

// NewWatcher creates a new event watcher with the specified configuration.
//
// The watcher is initialized with a 1000-event buffer that does not drop
// events when full. Events are buffered internally and made available through
// the Consume() method.
//
// The returned watcher is safe for concurrent use, but Watch() and Sync()
// should typically be run in separate goroutines.
func NewWatcher[T any](cfg WatcherConfig[T]) *Watcher[T] {
	w := &Watcher[T]{
		logger:       cfg.Logger,
		instanceID:   cfg.InstanceID,
		region:       cfg.Region,
		createStream: cfg.CreateStream,
	}

	return w
}

// Sync performs periodic full synchronization by creating synthetic streams.
//
// This method runs indefinitely in the background, creating a new stream every
// minute to fetch the complete current state. Synthetic streams (Synthetic=true,
// Live=false) return all current state rather than just live updates.
//
// The method handles connection errors gracefully by logging them and continuing
// with the next sync cycle. Each stream is properly closed after processing.
//
// This should be run in a separate goroutine:
//
//	go watcher.Sync(ctx)
func (w *Watcher[T]) Sync(ctx context.Context, buf *buffer.Buffer[*T]) {
	w.logger.Info("Starting Sync")

	req := connect.NewRequest(&ctrlv1.WatchRequest{
		ClientId: w.instanceID,
		Selectors: map[string]string{
			"region": w.region,
		},
		Synthetic: true,
		Live:      false,
	})
	repeat.Every(time.Minute, func() {
		stream, err := w.createStream(ctx, req)
		if err != nil {
			w.logger.Error("unable to connect to control plane", "error", err)
			return
		}
		for stream.Receive() {
			buf.Buffer(stream.Msg())
		}
		err = stream.Close()
		if err != nil {
			w.logger.Error("unable to close stream", "error", err)
		}

	})

}

// Watch establishes a live streaming connection to receive real-time events.
//
// This method runs indefinitely, maintaining a persistent connection to the
// control plane for live event delivery. It implements exponential backoff
// with jitter for reconnection attempts, with a maximum of 60 seconds between
// retries.
//
// The watcher handles connection failures gracefully:
// - Logs connection errors with failure count
// - Implements progressive backoff (1s, 2s, 3s... up to 60s)
// - Automatically reconnects when streams end
// - Resets failure counter on successful connections
//
// Live streams (Synthetic=false, Live=true) deliver only new events as they occur,
// not historical state. Use Sync() for full state reconciliation.
//
// This should be run in a separate goroutine:
//
//	go watcher.Watch(ctx)
func (w *Watcher[T]) Watch(ctx context.Context, buf *buffer.Buffer[*T]) {
	w.logger.Info("Starting Watch")

	req := connect.NewRequest(&ctrlv1.WatchRequest{
		ClientId: w.instanceID,
		Selectors: map[string]string{
			"region": w.region,
		},
		Synthetic: false,
		Live:      true,
	})
	go func() {

		consecutiveFailures := 0
		var stream *connect.ServerStreamForClient[T]
		var err error
		for {
			if stream == nil {
				stream, err = w.createStream(ctx, req)
				if err != nil {
					consecutiveFailures++
					w.logger.Error("unable to connect to control plane", "consecutive_failures", consecutiveFailures)
					time.Sleep(time.Duration(min(60, consecutiveFailures)) * time.Second)
					continue
				} else {
					consecutiveFailures = 0
				}
			}

			hasMsg := stream.Receive()
			if !hasMsg {
				w.logger.Debug("Stream ended, reconnecting...",
					"error", stream.Err(),
				)
				stream = nil
				time.Sleep(time.Second)
				continue
			}
			buf.Buffer(stream.Msg())
		}
	}()

}
