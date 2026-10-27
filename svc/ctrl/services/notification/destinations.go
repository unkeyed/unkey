package notification

import (
	"context"
	"database/sql"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/proto"
)

func (s *Service) ListDestinations(ctx context.Context, req *connect.Request[ctrlv1.NotificationListDestinationsRequest]) (*connect.Response[ctrlv1.NotificationListDestinationsResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	limit, err := pageLimit(req.Msg.GetLimit(), req.Msg.GetCursor())
	if err != nil {
		return nil, err
	}
	rows, err := s.db.ListNotificationDestinations(ctx, db.ListNotificationDestinationsParams{WorkspaceID: req.Msg.GetWorkspaceId(), PageCursor: req.Msg.GetCursor(), Limit: limit + 1})
	if err != nil {
		return nil, operationError(err)
	}
	response := &ctrlv1.NotificationListDestinationsResponse{HasMore: len(rows) > int(limit)}
	if response.HasMore {
		rows = rows[:limit]
		response.NextCursor = rows[len(rows)-1].ID
	}
	for _, row := range rows {
		destination, err := s.destinationDTO(ctx, row)
		if err != nil {
			return nil, err
		}
		response.Destinations = append(response.Destinations, destination)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) GetDestination(ctx context.Context, req *connect.Request[ctrlv1.NotificationGetDestinationRequest]) (*connect.Response[ctrlv1.NotificationGetDestinationResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	row, err := findDestination(ctx, s.db, req.Msg.GetWorkspaceId(), req.Msg.GetDestinationId())
	if err != nil {
		return nil, err
	}
	destination, err := s.destinationDTO(ctx, row)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationGetDestinationResponse{Destination: destination}), nil
}

func (s *Service) CreateDestination(ctx context.Context, req *connect.Request[ctrlv1.NotificationCreateDestinationRequest]) (*connect.Response[ctrlv1.NotificationCreateDestinationResponse], error) {
	message := req.Msg
	if err := s.authorize(ctx, req, message.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if !validID(message.GetIdempotencyKey(), 128) {
		return nil, rpcError(connect.CodeInvalidArgument, "idempotency_key is required and must be at most 128 bytes")
	}
	id := stableID(message.GetWorkspaceId(), "destination", message.GetIdempotencyKey())
	var destination *ctrlv1.NotificationDestination
	err := s.mutate(ctx, message.GetWorkspaceId(), message.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		row, err := queries.FindNotificationDestination(ctx, db.FindNotificationDestinationParams{WorkspaceID: message.GetWorkspaceId(), ID: id})
		if err == nil {
			destination, err = s.destinationDTO(ctx, row)
			return err
		}
		if !db.IsNotFound(err) {
			return err
		}
		if err := validateName(message.GetName()); err != nil {
			return err
		}
		var secret *string
		var headers *ctrlv1.NotificationHeaders
		if message.GetChannel() == "webhook" || message.GetSigningSecret() != "" {
			secret = proto.String(message.GetSigningSecret())
		}
		if len(message.GetHeaders()) > 0 {
			headers = &ctrlv1.NotificationHeaders{Items: message.GetHeaders()}
		}
		config := &notificationv1.DestinationConfig{}
		if err := s.configureDestination(ctx, message.GetWorkspaceId(), message.GetChannel(), config, proto.String(message.GetUrl()), headers, secret); err != nil {
			return err
		}
		encoded, err := proto.Marshal(config)
		if err != nil {
			return err
		}
		if err := queries.InsertNotificationDestination(ctx, db.InsertNotificationDestinationParams{
			ID:          id,
			WorkspaceID: message.GetWorkspaceId(),
			Name:        message.GetName(),
			Channel:     db.NotificationDestinationsChannel(message.GetChannel()),
			Config:      encoded,
			Enabled:     message.GetEnabled(),
			CreatedAt:   s.clock.Now().UnixMilli(),
		}); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, message.GetWorkspaceId(), message.GetActor(), "destination", "create", id); err != nil {
			return err
		}
		row, err = findDestination(ctx, queries, message.GetWorkspaceId(), id)
		if err != nil {
			return err
		}
		destination, err = s.destinationDTO(ctx, row)
		return err
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationCreateDestinationResponse{Destination: destination}), nil
}

func (s *Service) UpdateDestination(ctx context.Context, req *connect.Request[ctrlv1.NotificationUpdateDestinationRequest]) (*connect.Response[ctrlv1.NotificationUpdateDestinationResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	destination, err := s.updateDestination(ctx, req.Msg, "update")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationUpdateDestinationResponse{Destination: destination}), nil
}

func (s *Service) RotateSigningSecret(ctx context.Context, req *connect.Request[ctrlv1.NotificationRotateSigningSecretRequest]) (*connect.Response[ctrlv1.NotificationRotateSigningSecretResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	destination, err := s.updateDestination(ctx, &ctrlv1.NotificationUpdateDestinationRequest{
		WorkspaceId:   req.Msg.GetWorkspaceId(),
		DestinationId: req.Msg.GetDestinationId(),
		SigningSecret: proto.String(req.Msg.GetSigningSecret()),
		Actor:         req.Msg.GetActor(),
	}, "rotate_signing_secret")
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationRotateSigningSecretResponse{Destination: destination}), nil
}

func (s *Service) DeleteDestination(ctx context.Context, req *connect.Request[ctrlv1.NotificationDeleteDestinationRequest]) (*connect.Response[ctrlv1.NotificationDeleteDestinationResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	err := s.mutate(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		if _, err := findDestination(ctx, queries, req.Msg.GetWorkspaceId(), req.Msg.GetDestinationId()); err != nil {
			return err
		}
		referenced, err := queries.NotificationDestinationReferenced(ctx, db.NotificationDestinationReferencedParams{WorkspaceID: req.Msg.GetWorkspaceId(), ID: nullable(req.Msg.GetDestinationId())})
		if err != nil {
			return err
		}
		if referenced {
			return rpcError(connect.CodeFailedPrecondition, "destination is referenced by a notification rule")
		}
		if err := queries.DeleteNotificationDestination(ctx, db.DeleteNotificationDestinationParams{WorkspaceID: req.Msg.GetWorkspaceId(), ID: req.Msg.GetDestinationId()}); err != nil {
			return err
		}
		return s.audit(ctx, tx, req.Msg.GetWorkspaceId(), req.Msg.GetActor(), "destination", "delete", req.Msg.GetDestinationId())
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationDeleteDestinationResponse{}), nil
}

func (s *Service) updateDestination(ctx context.Context, message *ctrlv1.NotificationUpdateDestinationRequest, action string) (*ctrlv1.NotificationDestination, error) {
	var destination *ctrlv1.NotificationDestination
	err := s.mutate(ctx, message.GetWorkspaceId(), message.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		row, err := findDestination(ctx, queries, message.GetWorkspaceId(), message.GetDestinationId())
		if err != nil {
			return err
		}
		if message.Name != nil {
			if err := validateName(message.GetName()); err != nil {
				return err
			}
			row.Name = message.GetName()
		}
		if message.Enabled != nil {
			row.Enabled = message.GetEnabled()
		}
		config, err := decodeDestination(row)
		if err != nil {
			return err
		}
		if err := s.configureDestination(ctx, row.WorkspaceID, string(row.Channel), config, message.Url, message.Headers, message.SigningSecret); err != nil {
			return err
		}
		row.Config, err = proto.Marshal(config)
		if err != nil {
			return err
		}
		if err := queries.UpdateNotificationDestination(ctx, db.UpdateNotificationDestinationParams{
			WorkspaceID: row.WorkspaceID,
			ID:          row.ID,
			Name:        row.Name,
			Config:      row.Config,
			Enabled:     row.Enabled,
			UpdatedAt:   sql.NullInt64{Int64: s.clock.Now().UnixMilli(), Valid: true},
		}); err != nil {
			return err
		}
		if err := s.audit(ctx, tx, row.WorkspaceID, message.GetActor(), "destination", action, row.ID); err != nil {
			return err
		}
		destination, err = s.destinationDTO(ctx, row)
		return err
	})
	return destination, err
}

func findDestination(ctx context.Context, queries db.Querier, workspaceID, id string) (db.NotificationDestination, error) {
	if !validID(id, 48) {
		return db.NotificationDestination{}, rpcError(connect.CodeInvalidArgument, "destination_id is required")
	}
	row, err := queries.FindNotificationDestination(ctx, db.FindNotificationDestinationParams{WorkspaceID: workspaceID, ID: id})
	return row, operationError(err)
}
