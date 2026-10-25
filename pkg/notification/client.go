package notification

import (
	"context"
	"errors"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
)

const ServiceName = "notification.v1.NotificationService"
const DeliveryServiceName = "notification.v1.DeliveryService"

type Client struct {
	ingress *ingress.Client
}

func NewClient(client *ingress.Client) (*Client, error) {
	if client == nil {
		return nil, errors.New("notification ingress is required")
	}
	return &Client{ingress: client}, nil
}

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
