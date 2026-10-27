package notification

import (
	"errors"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
)

// Integration dispatches through the caller’s journal to stateful workers.
// It is not a Provider: returning nil does not assert external delivery.
type Integration interface {
	Name() string
	Accepts(eventType string) bool
	Dispatch(restate.Context, notification.Event) error
}

// DispatcherConfig selects integrations and optionally enables subscription delivery.
type DispatcherConfig struct {
	Definitions     []notification.Definition
	Integrations    []Integration
	DeliveryEnabled bool
}

// Dispatcher validates events before enqueueing subscription delivery and calling integrations.
type Dispatcher struct {
	definitions     map[string]notification.Definition
	integrations    []Integration
	deliveryEnabled bool
}

// NewDispatcher rejects duplicate event types and integration names.
func NewDispatcher(cfg DispatcherConfig) (*Dispatcher, error) {
	d := &Dispatcher{
		definitions:     make(map[string]notification.Definition),
		integrations:    append([]Integration(nil), cfg.Integrations...),
		deliveryEnabled: cfg.DeliveryEnabled,
	}
	for _, definition := range cfg.Definitions {
		if definition == nil || definition.Type() == "" || d.definitions[definition.Type()] != nil {
			return nil, errors.New("invalid or duplicate notification definition")
		}
		d.definitions[definition.Type()] = definition
	}
	names := make(map[string]bool)
	for _, integration := range d.integrations {
		if integration == nil || integration.Name() == "" || names[integration.Name()] {
			return nil, errors.New("invalid or duplicate notification integration")
		}
		names[integration.Name()] = true
	}
	return d, nil
}

// Dispatch journals subscription enqueue before integrations, which run in configured order.
// Call it from a Restate handler, outside Run closures. A result is not a delivery receipt.
func (d *Dispatcher) Dispatch(ctx restate.Context, event notification.Event) (string, error) {
	event, err := event.Normalize()
	if err != nil {
		return "", invalid("invalid notification event")
	}
	definition, ok := d.definitions[event.Type]
	if !ok {
		return "", invalid("unregistered notification type")
	}
	if _, err := definition.Render(event); err != nil {
		return "", invalid("invalid notification payload")
	}
	if d.deliveryEnabled {
		if _, err := notification.Send(ctx, event); err != nil {
			return "", err
		}
	}
	for _, integration := range d.integrations {
		if integration.Accepts(event.Type) {
			if err := integration.Dispatch(ctx, event); err != nil {
				return "", err
			}
		}
	}
	return event.ID(), nil
}
