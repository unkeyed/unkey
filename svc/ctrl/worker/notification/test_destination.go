package notification

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestDestination queues one test of a saved destination. Repeating the same key
// returns its current status without creating another provider delivery.
func (s *Service) TestDestination(ctx restate.ObjectContext, req *hydrav1.TestNotificationDestinationRequest) (*hydrav1.TestNotificationDestinationResponse, error) {
	if req.GetWorkspaceId() == "" || len(req.GetWorkspaceId()) > 48 || req.GetDestinationId() == "" || len(req.GetDestinationId()) > 48 ||
		req.GetIdempotencyKey() == "" || len(req.GetIdempotencyKey()) > 128 || strings.TrimSpace(req.GetIdempotencyKey()) != req.GetIdempotencyKey() ||
		strings.ContainsRune(req.GetIdempotencyKey(), '\x00') || !utf8.ValidString(req.GetIdempotencyKey()) {
		return nil, invalid("invalid destination test request")
	}
	if restate.Key(ctx) != notification.DestinationTestID(req.GetWorkspaceId(), req.GetDestinationId(), req.GetIdempotencyKey()) {
		return nil, invalid("invalid destination test key")
	}
	result, err := restate.Run(ctx, func(runCtx restate.RunContext) (*hydrav1.TestNotificationDestinationResponse, error) {
		return s.planTest(runCtx, req)
	}, restate.WithProtoJSON, restate.WithName("freeze destination test"))
	if err != nil {
		return nil, err
	}
	if result.GetStatus() == string(db.NotificationDeliveriesStatusPending) || result.GetStatus() == string(db.NotificationDeliveriesStatusSending) {
		ref := deliveryRef{WorkspaceID: req.GetWorkspaceId(), ID: result.GetDeliveryId()}
		restate.WithRequestType[deliveryRef](restate.Object[string](ctx, notification.DeliveryServiceName, ref.ID, "Deliver")).Send(ref, restate.WithIdempotencyKey(ref.ID))
	}
	return result, nil
}

func (s *Service) planTest(ctx context.Context, req *hydrav1.TestNotificationDestinationRequest) (*hydrav1.TestNotificationDestinationResponse, error) {
	destination, err := s.db.FindNotificationDestination(ctx, db.FindNotificationDestinationParams{
		WorkspaceID: req.GetWorkspaceId(),
		ID:          req.GetDestinationId(),
	})
	if db.IsNotFound(err) {
		return nil, restate.ToTerminalError(errors.New("destination not found"), restate.WithErrorCode(404))
	}
	if err != nil {
		return nil, err
	}
	event := notification.Event{
		WorkspaceID:   req.GetWorkspaceId(),
		SourceEventID: notification.DestinationTestID(req.GetWorkspaceId(), req.GetDestinationId(), req.GetIdempotencyKey()),
		Type:          "notification.test",
		SchemaVersion: 1,
		Resource:      notification.Resource{ProjectID: "", AppID: "", EnvironmentID: ""},
		Payload:       nil,
		OccurredAt:    0,
		ExpiresAt:     0,
	}
	channel := notification.Channel(destination.Channel)
	deliveryID := notification.DeliveryID(event.ID(), channel, "destination:"+destination.ID)
	previous, err := s.db.FindNotificationDelivery(ctx, db.FindNotificationDeliveryParams{WorkspaceID: req.GetWorkspaceId(), ID: deliveryID})
	if err == nil {
		return &hydrav1.TestNotificationDestinationResponse{DeliveryId: previous.ID, Status: string(previous.Status), Reason: previous.Reason.String}, nil
	}
	if !db.IsNotFound(err) {
		return nil, err
	}

	now := s.clock.Now()
	event.OccurredAt = now.UnixMilli()
	event.ExpiresAt = now.Add(5 * time.Minute).UnixMilli()
	event.Payload, err = json.Marshal(struct {
		DestinationID string `json:"destinationId"`
	}{DestinationID: destination.ID})
	if err != nil {
		return nil, err
	}
	hash, err := event.Fingerprint()
	if err != nil {
		return nil, err
	}
	target := recipient{key: "destination:" + destination.ID, channel: channel, config: destination.Config, email: "", reason: ""}
	if !destination.Enabled {
		target.reason = "destination_paused"
	} else if !validDestination(channel, destination.Config) {
		target.reason = "invalid_destination_config"
	}
	row, err := s.freezeDelivery(ctx, event, notification.Content{
		Subject: "Unkey test notification",
		Text:    "This is a test notification from Unkey. Your notification destination is connected.",
		HTML:    "",
	}, target, now.UnixMilli())
	if err != nil {
		return nil, err
	}
	if err := s.store(ctx, event, hash, []db.InsertNotificationDeliveryParams{row}, now.UnixMilli()); err != nil {
		return nil, err
	}
	return &hydrav1.TestNotificationDestinationResponse{DeliveryId: row.ID, Status: string(row.Status), Reason: row.Reason.String}, nil
}
