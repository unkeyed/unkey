package notification

import (
	"context"
	"database/sql"
	"strconv"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const targetsMax = 100

func (s *Service) ListRules(ctx context.Context, req *connect.Request[ctrlv1.NotificationListRulesRequest]) (*connect.Response[ctrlv1.NotificationListRulesResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	limit, err := pageLimit(req.Msg.GetLimit(), req.Msg.GetCursor())
	if err != nil {
		return nil, err
	}
	response := &ctrlv1.NotificationListRulesResponse{}
	err = db.Tx(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		rows, err := queries.ListNotificationRules(ctx, db.ListNotificationRulesParams{WorkspaceID: req.Msg.GetWorkspaceId(), PageCursor: req.Msg.GetCursor(), Limit: limit + 1})
		if err != nil {
			return err
		}
		response.HasMore = len(rows) > int(limit)
		if response.HasMore {
			rows = rows[:limit]
			response.NextCursor = rows[len(rows)-1].ID
		}
		for _, row := range rows {
			rule, err := ruleDTO(ctx, queries, row)
			if err != nil {
				return err
			}
			response.Rules = append(response.Rules, rule)
		}
		return nil
	})
	if err != nil {
		return nil, operationError(err)
	}
	return connect.NewResponse(response), nil
}

func (s *Service) GetRule(ctx context.Context, req *connect.Request[ctrlv1.NotificationGetRuleRequest]) (*connect.Response[ctrlv1.NotificationGetRuleResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	var rule *ctrlv1.NotificationRule
	err := db.Tx(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		var err error
		rule, err = findRule(ctx, db.NewQueries(tx), req.Msg.GetWorkspaceId(), req.Msg.GetRuleId())
		return err
	})
	if err != nil {
		return nil, operationError(err)
	}
	return connect.NewResponse(&ctrlv1.NotificationGetRuleResponse{Rule: rule}), nil
}

func (s *Service) CreateRule(ctx context.Context, req *connect.Request[ctrlv1.NotificationCreateRuleRequest]) (*connect.Response[ctrlv1.NotificationCreateRuleResponse], error) {
	message := req.Msg
	if err := s.authorize(ctx, req, message.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if !validID(message.GetIdempotencyKey(), 128) {
		return nil, rpcError(connect.CodeInvalidArgument, "idempotency_key is required and must be at most 128 bytes")
	}
	id := stableID(message.GetWorkspaceId(), "rule", message.GetIdempotencyKey())
	rule := &ctrlv1.NotificationRule{Id: id, Name: message.GetName(), NotificationType: message.GetNotificationType(), ProjectId: message.GetProjectId(), AppId: message.GetAppId(), EnvironmentId: message.GetEnvironmentId(), Enabled: message.GetEnabled(), Targets: message.GetTargets()}
	err := s.mutate(ctx, message.GetWorkspaceId(), message.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		row, err := queries.FindNotificationRule(ctx, db.FindNotificationRuleParams{WorkspaceID: message.GetWorkspaceId(), ID: id})
		if err == nil {
			rule, err = ruleDTO(ctx, queries, row)
			return err
		}
		if !db.IsNotFound(err) {
			return err
		}
		targets, err := s.validateRule(ctx, queries, message.GetWorkspaceId(), rule)
		if err != nil {
			return err
		}
		if err := queries.InsertNotificationRule(ctx, db.InsertNotificationRuleParams{
			ID:               id,
			WorkspaceID:      message.GetWorkspaceId(),
			Name:             rule.GetName(),
			NotificationType: rule.GetNotificationType(),
			ProjectID:        nullable(rule.GetProjectId()),
			AppID:            nullable(rule.GetAppId()),
			EnvironmentID:    nullable(rule.GetEnvironmentId()),
			Enabled:          rule.GetEnabled(),
			CreatedAt:        s.clock.Now().UnixMilli(),
		}); err != nil {
			return err
		}
		for _, target := range targets {
			if err := queries.InsertNotificationRuleTarget(ctx, target); err != nil {
				return err
			}
		}
		return s.audit(ctx, tx, message.GetWorkspaceId(), message.GetActor(), "rule", "create", id)
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationCreateRuleResponse{Rule: rule}), nil
}

func (s *Service) UpdateRule(ctx context.Context, req *connect.Request[ctrlv1.NotificationUpdateRuleRequest]) (*connect.Response[ctrlv1.NotificationUpdateRuleResponse], error) {
	message := req.Msg
	if err := s.authorize(ctx, req, message.GetWorkspaceId()); err != nil {
		return nil, err
	}
	rule := &ctrlv1.NotificationRule{Id: message.GetRuleId(), Name: message.GetName(), NotificationType: message.GetNotificationType(), ProjectId: message.GetProjectId(), AppId: message.GetAppId(), EnvironmentId: message.GetEnvironmentId(), Enabled: message.GetEnabled(), Targets: message.GetTargets()}
	err := s.mutate(ctx, message.GetWorkspaceId(), message.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		if _, err := findRule(ctx, queries, message.GetWorkspaceId(), rule.GetId()); err != nil {
			return err
		}
		targets, err := s.validateRule(ctx, queries, message.GetWorkspaceId(), rule)
		if err != nil {
			return err
		}
		if err := queries.UpdateNotificationRule(ctx, db.UpdateNotificationRuleParams{
			WorkspaceID:      message.GetWorkspaceId(),
			ID:               rule.GetId(),
			Name:             rule.GetName(),
			NotificationType: rule.GetNotificationType(),
			ProjectID:        nullable(rule.GetProjectId()),
			AppID:            nullable(rule.GetAppId()),
			EnvironmentID:    nullable(rule.GetEnvironmentId()),
			Enabled:          rule.GetEnabled(),
			UpdatedAt:        sql.NullInt64{Int64: s.clock.Now().UnixMilli(), Valid: true},
		}); err != nil {
			return err
		}
		if err := queries.DeleteNotificationRuleTargets(ctx, db.DeleteNotificationRuleTargetsParams{WorkspaceID: message.GetWorkspaceId(), SubscriptionID: rule.GetId()}); err != nil {
			return err
		}
		for _, target := range targets {
			if err := queries.InsertNotificationRuleTarget(ctx, target); err != nil {
				return err
			}
		}
		return s.audit(ctx, tx, message.GetWorkspaceId(), message.GetActor(), "rule", "update", rule.GetId())
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationUpdateRuleResponse{Rule: rule}), nil
}

func (s *Service) DeleteRule(ctx context.Context, req *connect.Request[ctrlv1.NotificationDeleteRuleRequest]) (*connect.Response[ctrlv1.NotificationDeleteRuleResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	err := s.mutate(ctx, req.Msg.GetWorkspaceId(), req.Msg.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		if _, err := findRule(ctx, queries, req.Msg.GetWorkspaceId(), req.Msg.GetRuleId()); err != nil {
			return err
		}
		if err := queries.DeleteNotificationRuleTargets(ctx, db.DeleteNotificationRuleTargetsParams{WorkspaceID: req.Msg.GetWorkspaceId(), SubscriptionID: req.Msg.GetRuleId()}); err != nil {
			return err
		}
		if err := queries.DeleteNotificationRule(ctx, db.DeleteNotificationRuleParams{WorkspaceID: req.Msg.GetWorkspaceId(), ID: req.Msg.GetRuleId()}); err != nil {
			return err
		}
		return s.audit(ctx, tx, req.Msg.GetWorkspaceId(), req.Msg.GetActor(), "rule", "delete", req.Msg.GetRuleId())
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationDeleteRuleResponse{}), nil
}

func (s *Service) validateRule(ctx context.Context, queries db.Querier, workspaceID string, rule *ctrlv1.NotificationRule) ([]db.InsertNotificationRuleTargetParams, error) {
	if err := validateName(rule.GetName()); err != nil {
		return nil, err
	}
	if err := validateType(rule.GetNotificationType()); err != nil {
		return nil, err
	}
	if len(rule.GetTargets()) == 0 || len(rule.GetTargets()) > targetsMax {
		return nil, rpcError(connect.CodeInvalidArgument, "a rule must have between 1 and 100 targets")
	}
	if err := validateScope(ctx, queries, workspaceID, rule.GetProjectId(), rule.GetAppId(), rule.GetEnvironmentId()); err != nil {
		return nil, err
	}
	var userIDs []string
	targets := make([]db.InsertNotificationRuleTargetParams, 0, len(rule.GetTargets()))
	seen := make(map[string]bool)
	for index, target := range rule.GetTargets() {
		row := db.InsertNotificationRuleTargetParams{
			ID:             stableID(workspaceID, "target", rule.GetId()+"\x00"+strconv.Itoa(index)),
			WorkspaceID:    workspaceID,
			SubscriptionID: rule.GetId(),
			Channel:        db.NotificationSubscriptionTargetsChannelEmail,
			Audience:       sql.NullInt32{},
			UserID:         sql.NullString{},
			DestinationID:  sql.NullString{},
			CreatedAt:      s.clock.Now().UnixMilli(),
		}
		var key string
		switch recipient := target.GetTarget().(type) {
		case *ctrlv1.NotificationTarget_DestinationId:
			destination, err := findDestination(ctx, queries, workspaceID, recipient.DestinationId)
			if err != nil {
				return nil, err
			}
			if _, err := decodeDestination(destination); err != nil {
				return nil, rpcError(connect.CodeFailedPrecondition, "destination configuration is unavailable")
			}
			row.Channel = db.NotificationSubscriptionTargetsChannel(destination.Channel)
			row.DestinationID = nullable(destination.ID)
			key = "destination:" + destination.ID
		case *ctrlv1.NotificationTarget_UserId:
			if !validID(recipient.UserId, 256) {
				return nil, rpcError(connect.CodeInvalidArgument, "user_id is required")
			}
			row.UserID = nullable(recipient.UserId)
			userIDs = append(userIDs, recipient.UserId)
			key = "user:" + recipient.UserId
		case *ctrlv1.NotificationTarget_Audience:
			if recipient.Audience != notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS {
				return nil, rpcError(connect.CodeInvalidArgument, "unsupported notification audience")
			}
			row.Audience = sql.NullInt32{Int32: int32(recipient.Audience), Valid: true}
			key = "audience:workspace_admins"
		default:
			return nil, rpcError(connect.CodeInvalidArgument, "a target must select a destination, user, or audience")
		}
		if seen[key] {
			return nil, rpcError(connect.CodeInvalidArgument, "duplicate notification target")
		}
		seen[key] = true
		targets = append(targets, row)
	}
	if len(userIDs) > 0 {
		if err := s.requireMembers(ctx, queries, workspaceID, userIDs); err != nil {
			return nil, err
		}
	}
	return targets, nil
}

func findRule(ctx context.Context, queries db.Querier, workspaceID, id string) (*ctrlv1.NotificationRule, error) {
	if !validID(id, 48) {
		return nil, rpcError(connect.CodeInvalidArgument, "rule_id is required")
	}
	row, err := queries.FindNotificationRule(ctx, db.FindNotificationRuleParams{WorkspaceID: workspaceID, ID: id})
	if err != nil {
		return nil, operationError(err)
	}
	return ruleDTO(ctx, queries, row)
}

func ruleDTO(ctx context.Context, queries db.Querier, row db.NotificationSubscription) (*ctrlv1.NotificationRule, error) {
	targets, err := queries.ListNotificationRuleTargets(ctx, db.ListNotificationRuleTargetsParams{WorkspaceID: row.WorkspaceID, SubscriptionID: row.ID})
	if err != nil {
		return nil, operationError(err)
	}
	if len(targets) > targetsMax {
		return nil, rpcError(connect.CodeFailedPrecondition, "notification rule target limit exceeded")
	}
	rule := &ctrlv1.NotificationRule{Id: row.ID, Name: row.Name, NotificationType: row.NotificationType, ProjectId: row.ProjectID.String, AppId: row.AppID.String, EnvironmentId: row.EnvironmentID.String, Enabled: row.Enabled}
	for _, target := range targets {
		switch {
		case target.DestinationID.Valid:
			rule.Targets = append(rule.Targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_DestinationId{DestinationId: target.DestinationID.String}})
		case target.UserID.Valid:
			rule.Targets = append(rule.Targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_UserId{UserId: target.UserID.String}})
		case target.Audience.Valid:
			rule.Targets = append(rule.Targets, &ctrlv1.NotificationTarget{Target: &ctrlv1.NotificationTarget_Audience{Audience: notificationv1.Audience(target.Audience.Int32)}})
		}
	}
	return rule, nil
}
