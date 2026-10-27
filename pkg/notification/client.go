package notification

import (
	"context"
	"errors"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
)

// ServiceName is the Restate virtual object service for event enqueueing.
const ServiceName = "notification.v1.NotificationService"

// DeliveryServiceName is the Restate virtual object service for individual deliveries.
const DeliveryServiceName = "notification.v1.DeliveryService"

// Notifier acknowledges durable enqueue, not provider acceptance or delivery.
// Notify returns the stable notification ID on success.
type Notifier interface {
	Notify(context.Context, Event) (string, error)
}

// Client enqueues events through Restate ingress. Construct it with NewClient.
type Client struct {
	ingress *ingress.Client
}

// NewClient requires a non-nil ingress client and returns nil on error.
func NewClient(client *ingress.Client) (*Client, error) {
	if client == nil {
		return nil, errors.New("notification ingress is required")
	}
	return &Client{ingress: client}, nil
}

// Notify validates and enqueues an event, returning its stable ID or an empty string on error.
// Success acknowledges enqueueing, not provider acceptance or delivery.
func (c *Client) Notify(ctx context.Context, event Event) (string, error) {
	event, err := event.Normalize()
	if err != nil {
		return "", err
	}
	fingerprint, err := event.Fingerprint()
	if err != nil {
		return "", err
	}
	_, err = ingress.Object[Event, string](c.ingress, ServiceName, event.ID(), "Notify").Send(ctx, event, restate.WithIdempotencyKey(fingerprint))
	if err != nil {
		return "", err
	}
	return event.ID(), nil
}

// Send journals the enqueue in an existing Restate handler. Do not call it from a Run closure.
func Send(ctx restate.Context, event Event) (string, error) {
	event, err := event.Normalize()
	if err != nil {
		return "", err
	}
	fingerprint, err := event.Fingerprint()
	if err != nil {
		return "", err
	}
	restate.Object[string](ctx, ServiceName, event.ID(), "Notify").Send(event, restate.WithIdempotencyKey(fingerprint))
	return event.ID(), nil
}

var _ Notifier = (*Client)(nil)
