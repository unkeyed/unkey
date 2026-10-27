package notification

import (
	"context"
	"errors"
	"sort"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/notificationrouting"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

func (s *Service) GetCatalog(ctx context.Context, req *connect.Request[ctrlv1.NotificationGetCatalogRequest]) (*connect.Response[ctrlv1.NotificationGetCatalogResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationGetCatalogResponse{
		EventTypes: eventTypes(),
		Channels:   []string{"email", "slack", "discord", "webhook"},
		Audiences:  []notificationv1.Audience{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS},
	}), nil
}

func (s *Service) ListMembers(ctx context.Context, req *connect.Request[ctrlv1.NotificationListMembersRequest]) (*connect.Response[ctrlv1.NotificationListMembersResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	limit, err := pageLimit(req.Msg.GetLimit(), req.Msg.GetCursor())
	if err != nil {
		return nil, err
	}
	if s.directory == nil {
		return nil, rpcError(connect.CodeFailedPrecondition, "member directory is not configured")
	}
	workspace, err := s.db.FindWorkspaceByID(ctx, req.Msg.GetWorkspaceId())
	if err != nil {
		return nil, operationError(err)
	}
	members, err := s.directory.Members(ctx, workspace.OrgID)
	if err != nil {
		return nil, directoryError(err)
	}
	response := &ctrlv1.NotificationListMembersResponse{}
	for _, member := range members {
		if member.UserID <= req.Msg.GetCursor() {
			continue
		}
		if len(response.Members) == int(limit) {
			response.HasMore = true
			response.NextCursor = response.Members[len(response.Members)-1].GetUserId()
			break
		}
		response.Members = append(response.Members, &ctrlv1.NotificationMember{UserId: member.UserID, Email: member.Email, Admin: member.Admin})
	}
	return connect.NewResponse(response), nil
}

func (s *Service) PreviewRecipients(ctx context.Context, req *connect.Request[ctrlv1.NotificationPreviewRecipientsRequest]) (*connect.Response[ctrlv1.NotificationPreviewRecipientsResponse], error) {
	message := req.Msg
	if err := s.authorize(ctx, req, message.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if err := validateType(message.GetNotificationType()); err != nil {
		return nil, err
	}
	if message.GetProjectId() == "" || message.GetAppId() == "" || message.GetEnvironmentId() == "" {
		return nil, rpcError(connect.CodeInvalidArgument, "preview requires a concrete project, app, and environment")
	}
	var recipients []notificationrouting.Recipient
	err := db.Tx(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		if err := validateScope(ctx, queries, message.GetWorkspaceId(), message.GetProjectId(), message.GetAppId(), message.GetEnvironmentId()); err != nil {
			return err
		}
		resolver := notificationrouting.Resolver{
			Database:  queries,
			Directory: s.directory,
			Audiences: map[notificationv1.Audience]notificationrouting.AudienceResolver{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS: notificationrouting.WorkspaceAdmins},
		}
		var err error
		recipients, err = resolver.Resolve(ctx, notification.Event{
			WorkspaceID:   message.GetWorkspaceId(),
			Type:          message.GetNotificationType(),
			Resource:      notification.Resource{ProjectID: message.GetProjectId(), AppID: message.GetAppId(), EnvironmentID: message.GetEnvironmentId()},
			SourceEventID: "",
			SchemaVersion: 0,
			Payload:       nil,
			OccurredAt:    0,
			ExpiresAt:     0,
		})
		return err
	})
	if err != nil {
		var invalid *notificationrouting.InvalidError
		if errors.As(err, &invalid) {
			return nil, rpcError(connect.CodeFailedPrecondition, invalid.Reason)
		}
		return nil, operationError(err)
	}
	response := &ctrlv1.NotificationPreviewRecipientsResponse{}
	for _, recipient := range recipients {
		item := &ctrlv1.NotificationRecipient{Key: recipient.Key, Channel: string(recipient.Channel), Email: recipient.Email, Reason: recipient.Reason}
		if recipient.Channel == notification.Email {
			item.UserId = strings.TrimPrefix(recipient.Key, "user:")
		} else {
			item.DestinationId = strings.TrimPrefix(recipient.Key, "destination:")
		}
		response.Recipients = append(response.Recipients, item)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) requireMembers(ctx context.Context, queries db.Querier, workspaceID string, userIDs []string) error {
	if s.directory == nil {
		return rpcError(connect.CodeFailedPrecondition, "member directory is not configured")
	}
	workspace, err := queries.FindWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return operationError(err)
	}
	members, err := s.directory.LookupMembers(ctx, workspace.OrgID, userIDs)
	if err != nil {
		return directoryError(err)
	}
	eligible := make([]string, 0, len(members))
	for _, member := range members {
		eligible = append(eligible, member.UserID)
	}
	sort.Strings(eligible)
	for _, userID := range userIDs {
		index := sort.SearchStrings(eligible, userID)
		if index == len(eligible) || eligible[index] != userID {
			return rpcError(connect.CodeInvalidArgument, "user is not an eligible verified workspace member")
		}
	}
	return nil
}

func directoryError(err error) error {
	if errors.Is(err, workos.ErrDirectoryLimit) {
		return rpcError(connect.CodeResourceExhausted, "member directory limit exceeded")
	}
	return rpcError(connect.CodeUnavailable, "member directory is unavailable")
}
