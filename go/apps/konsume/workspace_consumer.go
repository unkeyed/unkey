package konsume

import (
	"context"

	analyticsv1 "github.com/unkeyed/unkey/go/gen/proto/analytics/v1"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/eventstream"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// WorkspaceAwareConsumer wraps the Kafka consumer and routes events to
// workspace-specific writers via the WorkspaceWriterManager.
type WorkspaceAwareConsumer struct {
	keyVerificationConsumer eventstream.Consumer[*analyticsv1.KeyVerificationEvent]
	ratelimitConsumer       eventstream.Consumer[*analyticsv1.RatelimitEvent]
	apiRequestConsumer      eventstream.Consumer[*analyticsv1.ApiRequestEvent]
	writerManager           *WorkspaceWriterManager
	logger                  logging.Logger
}

// WorkspaceAwareConsumerConfig contains configuration for workspace-aware consumers
type WorkspaceAwareConsumerConfig struct {
	Brokers       []string
	Topics        Topics
	InstanceID    string
	WriterManager *WorkspaceWriterManager
	Logger        logging.Logger
}

// NewWorkspaceAwareConsumer creates a new workspace-aware Kafka consumer
func NewWorkspaceAwareConsumer(config WorkspaceAwareConsumerConfig) (*WorkspaceAwareConsumer, error) {
	// Create eventstream topics for each event type
	keyVerificationTopic := eventstream.NewTopic[*analyticsv1.KeyVerificationEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.KeyVerifications,
			InstanceID: config.InstanceID,
			Logger:     config.Logger,
		},
	)

	ratelimitTopic := eventstream.NewTopic[*analyticsv1.RatelimitEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.Ratelimits,
			InstanceID: config.InstanceID,
			Logger:     config.Logger,
		},
	)

	apiRequestTopic := eventstream.NewTopic[*analyticsv1.ApiRequestEvent](
		eventstream.TopicConfig{
			Brokers:    config.Brokers,
			Topic:      config.Topics.ApiRequests,
			InstanceID: config.InstanceID,
			Logger:     config.Logger,
		},
	)

	consumer := &WorkspaceAwareConsumer{
		keyVerificationConsumer: keyVerificationTopic.NewConsumer(eventstream.WithStartFromBeginning()),
		ratelimitConsumer:       ratelimitTopic.NewConsumer(eventstream.WithStartFromBeginning()),
		apiRequestConsumer:      apiRequestTopic.NewConsumer(eventstream.WithStartFromBeginning()),
		writerManager:           config.WriterManager,
		logger:                  config.Logger,
	}

	return consumer, nil
}

// Start begins consuming messages from Kafka topics
func (c *WorkspaceAwareConsumer) Start(ctx context.Context) error {
	c.logger.Info("starting workspace-aware kafka consumer")

	// Start consuming from each topic
	c.keyVerificationConsumer.Consume(ctx, c.handleKeyVerification)
	c.ratelimitConsumer.Consume(ctx, c.handleRatelimit)
	c.apiRequestConsumer.Consume(ctx, c.handleApiRequest)

	return nil
}

// Stop gracefully stops the consumer
func (c *WorkspaceAwareConsumer) Stop(ctx context.Context) error {
	c.logger.Info("stopping workspace-aware kafka consumer")

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

	// Close the writer manager
	if err := c.writerManager.Close(ctx); err != nil {
		c.logger.Error("error closing writer manager", "error", err.Error())
		if firstError == nil {
			firstError = err
		}
	}

	c.logger.Info("workspace-aware kafka consumer stopped")
	return firstError
}

// handleKeyVerification processes a key verification event from Kafka
func (c *WorkspaceAwareConsumer) handleKeyVerification(ctx context.Context, event *analyticsv1.KeyVerificationEvent) error {
	// Get workspace-specific writer
	writer, err := c.writerManager.GetWriter(ctx, event.WorkspaceId)
	if err != nil {
		c.logger.Error("failed to get writer for workspace",
			"workspace_id", event.WorkspaceId,
			"request_id", event.RequestId,
			"error", err.Error(),
		)

		return err
	}

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

	return writer.KeyVerification(ctx, data)
}

// handleRatelimit processes a ratelimit event from Kafka
func (c *WorkspaceAwareConsumer) handleRatelimit(ctx context.Context, event *analyticsv1.RatelimitEvent) error {
	c.logger.Info("processing ratelimit event", "event", event)

	// Get workspace-specific writer
	writer, err := c.writerManager.GetWriter(ctx, event.WorkspaceId)
	if err != nil {
		c.logger.Error("failed to get writer for workspace",
			"workspace_id", event.WorkspaceId,
			"request_id", event.RequestId,
			"error", err.Error(),
		)
		return err
	}

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

	return writer.Ratelimit(ctx, data)
}

// handleApiRequest processes an API request event from Kafka
func (c *WorkspaceAwareConsumer) handleApiRequest(ctx context.Context, event *analyticsv1.ApiRequestEvent) error {
	c.logger.Info("processing api request event", "event", event)

	// Get workspace-specific writer
	writer, err := c.writerManager.GetWriter(ctx, event.WorkspaceId)
	if err != nil {
		c.logger.Error("failed to get writer for workspace",
			"workspace_id", event.WorkspaceId,
			"request_id", event.RequestId,
			"error", err.Error(),
		)
		return err
	}

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

	return writer.ApiRequest(ctx, data)
}
