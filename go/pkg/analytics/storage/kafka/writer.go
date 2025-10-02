package kafka

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"

	analyticsv1 "github.com/unkeyed/unkey/go/gen/proto/analytics/v1"
)

// Writer implements the analytics.Writer interface for Kafka message publishing.
// It uses the existing eventstream infrastructure with protobuf serialization.
type Writer struct {
	keyVerificationProducer eventstream.Producer[*analyticsv1.KeyVerificationEvent]
	ratelimitProducer       eventstream.Producer[*analyticsv1.RatelimitEvent]
	apiRequestProducer      eventstream.Producer[*analyticsv1.ApiRequestEvent]
	logger                  logging.Logger
}

// New creates a new Kafka writer with the provided configuration.
func New(config Config, logger logging.Logger) (analytics.Writer, error) {
	if len(config.Brokers) == 0 {
		return nil, fault.New("kafka brokers are required")
	}

	// Create eventstream topics for each event type
	keyVerificationTopic := eventstream.NewTopic[*analyticsv1.KeyVerificationEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.KeyVerifications,
			InstanceID: "analytics",
			Logger:     logger,
		},
	)

	ratelimitTopic := eventstream.NewTopic[*analyticsv1.RatelimitEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.Ratelimits,
			InstanceID: "analytics",
			Logger:     logger,
		},
	)

	apiRequestTopic := eventstream.NewTopic[*analyticsv1.ApiRequestEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.ApiRequests,
			InstanceID: "analytics",
			Logger:     logger,
		},
	)

	writer := &Writer{
		keyVerificationProducer: keyVerificationTopic.NewProducer(),
		ratelimitProducer:       ratelimitTopic.NewProducer(),
		apiRequestProducer:      apiRequestTopic.NewProducer(),
		logger:                  logger,
	}

	return writer, nil
}

// KeyVerification publishes a key verification event to the configured Kafka topic.
func (w *Writer) KeyVerification(ctx context.Context, data schema.KeyVerificationV2) error {
	// Convert to protobuf event
	event := &analyticsv1.KeyVerificationEvent{
		RequestId:    data.RequestID,
		Time:         data.Time,
		WorkspaceId:  data.WorkspaceID,
		KeySpaceId:   data.KeySpaceID,
		IdentityId:   data.IdentityID,
		KeyId:        data.KeyID,
		Region:       data.Region,
		Outcome:      data.Outcome,
		Tags:         data.Tags,
		SpentCredits: data.SpentCredits,
		Latency:      data.Latency,
	}

	return w.keyVerificationProducer.Produce(ctx, event)
}

// Ratelimit publishes a ratelimit event to the configured Kafka topic.
func (w *Writer) Ratelimit(ctx context.Context, data schema.RatelimitV2) error {
	// Convert to protobuf event
	event := &analyticsv1.RatelimitEvent{
		RequestId:   data.RequestID,
		Time:        data.Time,
		WorkspaceId: data.WorkspaceID,
		NamespaceId: data.NamespaceID,
		Identifier:  data.Identifier,
		Passed:      data.Passed,
		Latency:     data.Latency,
		OverrideId:  data.OverrideID,
		Limit:       data.Limit,
		Remaining:   data.Remaining,
		ResetAt:     data.ResetAt,
	}

	return w.ratelimitProducer.Produce(ctx, event)
}

// ApiRequest publishes an API request event to the configured Kafka topic.
func (w *Writer) ApiRequest(ctx context.Context, data schema.ApiRequestV2) error {
	// Convert query params to protobuf format
	queryParams := make(map[string]*analyticsv1.QueryParams)
	for key, values := range data.QueryParams {
		queryParams[key] = &analyticsv1.QueryParams{
			Values: values,
		}
	}

	// Convert to protobuf event
	event := &analyticsv1.ApiRequestEvent{
		RequestId:       data.RequestID,
		Time:            data.Time,
		WorkspaceId:     data.WorkspaceID,
		Host:            data.Host,
		Method:          data.Method,
		Path:            data.Path,
		QueryString:     data.QueryString,
		QueryParams:     queryParams,
		RequestHeaders:  data.RequestHeaders,
		RequestBody:     data.RequestBody,
		ResponseStatus:  data.ResponseStatus,
		ResponseHeaders: data.ResponseHeaders,
		ResponseBody:    data.ResponseBody,
		Error:           data.Error,
		ServiceLatency:  data.ServiceLatency,
		UserAgent:       data.UserAgent,
		IpAddress:       data.IpAddress,
		Region:          data.Region,
	}

	return w.apiRequestProducer.Produce(ctx, event)
}

// Close gracefully shuts down all Kafka producers.
func (w *Writer) Close(ctx context.Context) error {
	w.logger.Info("closing kafka writer")

	var firstError error

	// Close all producers
	if err := w.keyVerificationProducer.Close(); err != nil {
		w.logger.Error("failed to close key verification producer", "error", err.Error())
		firstError = err
	}

	if err := w.ratelimitProducer.Close(); err != nil {
		w.logger.Error("failed to close ratelimit producer", "error", err.Error())
		if firstError == nil {
			firstError = err
		}
	}

	if err := w.apiRequestProducer.Close(); err != nil {
		w.logger.Error("failed to close api request producer", "error", err.Error())
		if firstError == nil {
			firstError = err
		}
	}

	return firstError
}
