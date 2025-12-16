package watcher

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/buffer"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

type Watcher[T any] struct {
	logger     logging.Logger
	events     *buffer.Buffer[*T]
	instanceID string
	region     string
	shard      string
}

type Config struct {
	Logger     logging.Logger
	Cluster    *ctrlv1connect.ClusterServiceClient
	InstanceID string
	Region     string
	Shard      string
}

func New[T any](cfg Config) *Watcher[T] {
	w := &Watcher[T]{
		logger: cfg.Logger,
		events: buffer.New[*T](buffer.Config{
			Capacity: 1000,
			Drop:     false,
			Name:     "krane_watcher",
		}),
		instanceID: cfg.InstanceID,
		region:     cfg.Region,
		shard:      cfg.Shard,
	}

	return w
}

func (w *Watcher[T]) Consume() <-chan *T {
	return w.events.Consume()
}
func (w *Watcher[T]) buildRequest() *connect.Request[ctrlv1.WatchRequest] {

	return connect.NewRequest(&ctrlv1.WatchRequest{
		ClientId: w.instanceID,
		Selectors: map[string]string{
			"region": w.region,
			"shard":  w.shard,
		},
	})

}

func (w *Watcher[T]) Sync(ctx context.Context, createStream CreateStream[T]) {
	go func() {

		req := w.buildRequest()
		repeat.Every(time.Minute, func() {
			stream, err := createStream(ctx, req)
			if err != nil {
				w.logger.Error("unable to connect to control plane", "error", err)
				return
			}
			for stream.Receive() {
				w.events.Buffer(stream.Msg())
			}
			err = stream.Close()
			if err != nil {
				w.logger.Error("unable to close stream", "error", err)
			}

		})
	}()

}

type CreateStream[T any] func(context.Context, *connect.Request[ctrlv1.WatchRequest]) (*connect.ServerStreamForClient[T], error)

func (w *Watcher[T]) Watch(ctx context.Context, createStream CreateStream[T]) {

	req := w.buildRequest()

	consecutiveFailures := 0
	var stream *connect.ServerStreamForClient[T]
	var err error
	for {
		if stream == nil {
			stream, err = createStream(ctx, req)
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
			w.logger.Info("Stream ended, reconnecting...",
				"error", stream.Err(),
			)
			stream = nil
			time.Sleep(time.Second)
			continue
		}
		w.events.Buffer(stream.Msg())
	}

}

//func (s *SyncEngine) newStream() (*connect.ServerStreamForClient[ctrlv1.InfraEvent], error) {
//	s.logger.Info("connecting to control plane to start stream")
//
//	stream, err := s.ctrl.Watch(context.Background(), connect.NewRequest(&ctrlv1.WatchRequest{
//		ClientId: s.instanceID,
//		Selectors: map[string]string{
//			"region": s.region,
//			"shard":  s.shard,
//		},
//	}))
//	s.logger.Info("stream", "stream", stream, "err", err)
//
//	return stream, err
//
//}
