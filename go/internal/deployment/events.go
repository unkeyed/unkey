package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisEventPublisher struct {
	client    *redis.Client
	streamKey string
	maxLen    int64
}

func NewRedisEventPublisher(client *redis.Client, streamKey string, maxLen int64) *RedisEventPublisher {
	if maxLen <= 0 {
		maxLen = 10000 // Default max length
	}
	return &RedisEventPublisher{
		client:    client,
		streamKey: streamKey,
		maxLen:    maxLen,
	}
}

func (p *RedisEventPublisher) Publish(ctx context.Context, event *DeploymentEvent) error {
	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Prepare Redis Stream entry
	values := map[string]interface{}{
		"type":          event.Type,
		"deployment_id": event.DeploymentID,
		"customer_id":   event.CustomerID,
		"project_id":    event.ProjectID,
		"status":        event.Status,
		"timestamp":     event.Timestamp.Format(time.RFC3339),
		"data":          string(eventData),
	}

	if event.Step != "" {
		values["step"] = event.Step
	}

	// Add to Redis Stream with automatic trimming
	args := &redis.XAddArgs{
		Stream: p.streamKey,
		MaxLen: p.maxLen,
		Approx: true,
		Values: values,
	}

	_, err = p.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to publish event to Redis stream: %w", err)
	}

	return nil
}

type WebhookEventPublisher struct {
	client    *redis
