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

// Consumer consumes analytics events from Kafka topics using the eventstream
// infrastructure and routes them to configured storage writers.
type Consumer struct {
	keyVerificationConsumer eventstream.Consumer[*analyticsv1.KeyVerificationEvent]
	ratelimitConsumer       eventstream.Consumer[*analyticsv1.RatelimitEvent]
	apiRequestConsumer      eventstream.Consumer[*analyticsv1.ApiRequestEvent]
	writers                 []analytics.Writer
	logger                  logging.Logger
}

// ConsumerConfig contains configuration for Kafka consumers.
type ConsumerConfig struct {
	// Kafka configuration
	Kafka Config `json:"kafka"`

	// Storage writers to route consumed events to
	Writers []analytics.Writer `json:"-"`

	// Logger for consumer operations
	Logger logging.Logger `json:"-"`
}

// NewConsumer creates a new Kafka consumer that routes events to storage writers.
func NewConsumer(config ConsumerConfig) (*Consumer, error) {
	if len(config.Kafka.Brokers) == 0 {
		return nil, fault.New("kafka brokers are required")
	}

	// Create eventstream topics for each event type
	keyVerificationTopic := eventstream.NewTopic[*analyticsv1.KeyVerificationEvent](
		eventstream.TopicConfig{
			Brokers:    config.Kafka.Brokers,
			Topic:      config.Kafka.Topics.KeyVerifications,
			InstanceID: "analytics-consumer",
			Logger:     config.Logger,
		},
	)

	ratelimitTopic := eventstream.NewTopic[*analyticsv1.RatelimitEvent](
		eventstream.TopicConfig{
			Brokers:    config.Kafka.Brokers,
			Topic:      config.Kafka.Topics.Ratelimits,
			InstanceID: "analytics-consumer",
			Logger:     config.Logger,
		},
	)

	apiRequestTopic := eventstream.NewTopic[*analyticsv1.ApiRequestEvent](
		eventstream.TopicConfig{
			Brokers:    config.Kafka.Brokers,
			Topic:      config.Kafka.Topics.ApiRequests,
			InstanceID: "analytics-consumer",
			Logger:     config.Logger,
		},
	)

	consumer := &Consumer{
		keyVerificationConsumer: keyVerificationTopic.NewConsumer(),
		ratelimitConsumer:       ratelimitTopic.NewConsumer(),
		apiRequestConsumer:      apiRequestTopic.NewConsumer(),
		writers:                 config.Writers,
		logger:                  config.Logger,
	}

	return consumer, nil
}

// Start begins consuming messages from Kafka topics and routing them to writers.
func (c *Consumer) Start(ctx context.Context) error {
	c.logger.Info("starting kafka consumer",
		"writers_count", len(c.writers),
	)

	// Start consuming from each topic
	c.keyVerificationConsumer.Consume(ctx, c.handleKeyVerification)
	c.ratelimitConsumer.Consume(ctx, c.handleRatelimit)
	c.apiRequestConsumer.Consume(ctx, c.handleApiRequest)

	return nil
}

// Stop gracefully stops the consumer and closes all writers.
func (c *Consumer) Stop(ctx context.Context) error {
	c.logger.Info("stopping kafka consumer")

	var firstError error

	// Close all consumers
	if err := c.keyVerificationConsumer.Close(); err != nil {
		c.logger.Error("error closing key verification consumer", "error", err.Error())
		firstError = err
	}

	if err := c.ratelimitConsumer.Close(); err != nil {
		c.logger.Error("error closing ratelimit consumer", "error", err.Error())
		if firstError == nil {
			firstError = err
		}
	}

	if err := c.apiRequestConsumer.Close(); err != nil {
		c.logger.Error("error closing api request consumer", "error", err.Error())
		if firstError == nil {
			firstError = err
		}
	}

	// Close all writers
	for i, writer := range c.writers {
		if err := writer.Close(ctx); err != nil {
			c.logger.Error("error closing writer",
				"writer_index", i,
				"error", err.Error(),
			)
			if firstError == nil {
				firstError = err
			}
		}
	}

	c.logger.Info("kafka consumer stopped")
	return firstError
}

// handleKeyVerification processes a key verification event from Kafka.
func (c *Consumer) handleKeyVerification(ctx context.Context, event *analyticsv1.KeyVerificationEvent) error {
	// Convert from protobuf to schema
	data := schema.KeyVerificationV2{
		RequestID:    event.RequestId,
		Time:         event.Time,
		WorkspaceID:  event.WorkspaceId,
		KeySpaceID:   event.KeySpaceId,
		IdentityID:   event.IdentityId,
		KeyID:        event.KeyId,
		Region:       event.Region,
		Outcome:      event.Outcome,
		Tags:         event.Tags,
		SpentCredits: event.SpentCredits,
		Latency:      event.Latency,
	}

	return c.routeToWriters(ctx, func(writer analytics.Writer) error {
		return writer.KeyVerification(ctx, data)
	})
}

// handleRatelimit processes a ratelimit event from Kafka.
func (c *Consumer) handleRatelimit(ctx context.Context, event *analyticsv1.RatelimitEvent) error {
	// Convert from protobuf to schema
	data := schema.RatelimitV2{
		RequestID:   event.RequestId,
		Time:        event.Time,
		WorkspaceID: event.WorkspaceId,
		NamespaceID: event.NamespaceId,
		Identifier:  event.Identifier,
		Passed:      event.Passed,
		Latency:     event.Latency,
		OverrideID:  event.OverrideId,
		Limit:       event.Limit,
		Remaining:   event.Remaining,
		ResetAt:     event.ResetAt,
	}

	return c.routeToWriters(ctx, func(writer analytics.Writer) error {
		return writer.Ratelimit(ctx, data)
	})
}

// handleApiRequest processes an API request event from Kafka.
func (c *Consumer) handleApiRequest(ctx context.Context, event *analyticsv1.ApiRequestEvent) error {
	// Convert query params from protobuf format
	queryParams := make(map[string][]string)
	for key, params := range event.QueryParams {
		queryParams[key] = params.Values
	}

	// Convert from protobuf to schema
	data := schema.ApiRequestV2{
		RequestID:       event.RequestId,
		Time:            event.Time,
		WorkspaceID:     event.WorkspaceId,
		Host:            event.Host,
		Method:          event.Method,
		Path:            event.Path,
		QueryString:     event.QueryString,
		QueryParams:     queryParams,
		RequestHeaders:  event.RequestHeaders,
		RequestBody:     event.RequestBody,
		ResponseStatus:  event.ResponseStatus,
		ResponseHeaders: event.ResponseHeaders,
		ResponseBody:    event.ResponseBody,
		Error:           event.Error,
		ServiceLatency:  event.ServiceLatency,
		UserAgent:       event.UserAgent,
		IpAddress:       event.IpAddress,
		Region:          event.Region,
	}

	return c.routeToWriters(ctx, func(writer analytics.Writer) error {
		return writer.ApiRequest(ctx, data)
	})
}

// routeToWriters executes the write function on all configured writers.
func (c *Consumer) routeToWriters(ctx context.Context, writeFunc func(analytics.Writer) error) error {
	var firstError error

	for i, writer := range c.writers {
		if err := writeFunc(writer); err != nil {
			c.logger.Error("writer failed",
				"writer_index", i,
				"error", err.Error(),
			)
			if firstError == nil {
				firstError = err
			}
		}
	}

	return firstError
}
