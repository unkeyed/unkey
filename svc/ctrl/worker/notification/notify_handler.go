package notification

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
)

// Notify freezes routes once per event identity and queues pending deliveries.
// Its result confirms planning, not provider acceptance.
func (s *Service) Notify(ctx restate.ObjectContext, event notification.Event) (string, error) {
	if err := event.Validate(); err != nil {
		return "", invalid(err.Error())
	}
	if restate.Key(ctx) != event.ID() {
		return "", invalid("notification key does not match event")
	}

	deliveries, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]deliveryRef, error) {
		return s.plan(runCtx, event)
	}, restate.WithName("freeze notification routing"))
	if err != nil {
		return "", err
	}
	for _, delivery := range deliveries {
		restate.Object[string](ctx, notification.DeliveryServiceName, delivery.ID, "Deliver").Send(delivery, restate.WithIdempotencyKey(delivery.ID))
	}
	return event.ID(), nil
}

func invalid(reason string) error {
	return restate.ToTerminalError(fmt.Errorf("notification: %s", reason), restate.WithErrorCode(400))
}
