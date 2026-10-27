package notificationrouting

import (
	"context"
	"database/sql"
	"errors"
	"sort"

	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"google.golang.org/protobuf/proto"
)

const recipientsMax = 1000

type Recipient struct {
	Key     string
	Channel notification.Channel
	Email   string
	Config  []byte
	Reason  string
}

type AudienceResolver func(context.Context, string, []workos.Member) ([]string, error)

type Resolver struct {
	Database  db.Querier
	Directory workos.Directory
	Audiences map[notificationv1.Audience]AudienceResolver
}

type InvalidError struct {
	Reason string
}

func (e *InvalidError) Error() string { return e.Reason }

func invalid(reason string) error { return &InvalidError{Reason: reason} }

// WorkspaceAdmins resolves the built-in admin audience from verified membership.
func WorkspaceAdmins(_ context.Context, _ string, members []workos.Member) ([]string, error) {
	ids := make([]string, 0)
	for _, member := range members {
		if member.Admin {
			ids = append(ids, member.UserID)
		}
	}
	return ids, nil
}

func (s Resolver) Resolve(ctx context.Context, event notification.Event) ([]Recipient, error) {
	targets, err := s.Database.ListNotificationTargets(ctx, db.ListNotificationTargetsParams{
		WorkspaceID:      event.WorkspaceID,
		NotificationType: event.Type,
		ProjectID:        nullable(event.Resource.ProjectID),
		AppID:            nullable(event.Resource.AppID),
		EnvironmentID:    nullable(event.Resource.EnvironmentID),
	})
	if err != nil {
		return nil, err
	}
	if len(targets) > recipientsMax {
		return nil, invalid("notification target limit exceeded")
	}

	needsMembers := false
	needsAudience := false
	var userIDs []string
	for _, target := range targets {
		if target.Channel != db.NotificationSubscriptionTargetsChannelEmail {
			continue
		}
		needsMembers = true
		needsAudience = needsAudience || target.Audience.Valid
		if target.UserID.Valid {
			userIDs = append(userIDs, target.UserID.String)
		}
	}
	var members []workos.Member
	var preferences []db.ListNotificationPreferencesRow
	if needsMembers {
		if s.Directory == nil {
			return nil, invalid("notification member directory is not configured")
		}
		workspace, err := s.Database.FindWorkspaceByID(ctx, event.WorkspaceID)
		if err != nil {
			return nil, err
		}
		if needsAudience {
			members, err = s.Directory.Members(ctx, workspace.OrgID)
		} else {
			members, err = s.Directory.LookupMembers(ctx, workspace.OrgID, userIDs)
		}
		if errors.Is(err, workos.ErrDirectoryLimit) {
			return nil, invalid("notification member directory limit exceeded")
		}
		if err != nil {
			return nil, err
		}
		preferences, err = s.Database.ListNotificationPreferences(ctx, db.ListNotificationPreferencesParams{WorkspaceID: event.WorkspaceID, NotificationType: event.Type})
		if err != nil {
			return nil, err
		}
		if len(preferences) > 10000 {
			return nil, invalid("notification preference limit exceeded")
		}
	}
	audiences := make(map[notificationv1.Audience][]string)
	for _, target := range targets {
		if !target.Audience.Valid {
			continue
		}
		audience := notificationv1.Audience(target.Audience.Int32)
		if _, found := audiences[audience]; found {
			continue
		}
		resolver := s.Audiences[audience]
		if resolver == nil {
			return nil, invalid("notification audience has no resolver")
		}
		ids, err := resolver(ctx, event.WorkspaceID, members)
		if err != nil {
			return nil, err
		}
		audiences[audience] = ids
	}
	return resolveTargets(targets, members, preferences, audiences)
}

func resolveTargets(targets []db.ListNotificationTargetsRow, members []workos.Member, preferences []db.ListNotificationPreferencesRow, audiences map[notificationv1.Audience][]string) ([]Recipient, error) {
	memberByID := make(map[string]workos.Member, len(members))
	for _, member := range members {
		memberByID[member.UserID] = member
	}
	optedOut := make(map[string]bool, len(preferences))
	for _, preference := range preferences {
		optedOut[preference.UserID] = !preference.Enabled
	}
	unique := make(map[string]Recipient)
	for _, target := range targets {
		channel := notification.Channel(target.Channel)
		if channel != notification.Email {
			key := "destination:" + target.DestinationID.String
			reason := ""
			switch {
			case !target.DestinationEnabled.Valid:
				reason = "destination_unavailable"
			case !target.DestinationEnabled.Bool:
				reason = "destination_paused"
			case !ValidDestination(channel, []byte(target.DestinationConfig.String)):
				reason = "invalid_destination_config"
			}
			unique[string(channel)+":"+key] = Recipient{Key: key, Channel: channel, Email: "", Config: []byte(target.DestinationConfig.String), Reason: reason}
			continue
		}
		ids := []string{target.UserID.String}
		if target.Audience.Valid {
			ids = audiences[notificationv1.Audience(target.Audience.Int32)]
		}
		for _, userID := range ids {
			member, found := memberByID[userID]
			reason := ""
			if !found {
				reason = "member_unavailable"
			} else if optedOut[userID] {
				reason = "email_opted_out"
			}
			key := "user:" + userID
			unique["email:"+key] = Recipient{Key: key, Channel: channel, Email: member.Email, Config: nil, Reason: reason}
		}
	}
	if len(unique) > recipientsMax {
		return nil, invalid("notification recipient limit exceeded")
	}
	result := make([]Recipient, 0, len(unique))
	for _, target := range unique {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i].Channel)+result[i].Key < string(result[j].Channel)+result[j].Key
	})
	return result, nil
}

func ValidDestination(channel notification.Channel, data []byte) bool {
	if len(data) > 64*1024 {
		return false
	}
	config := &notificationv1.DestinationConfig{}
	if proto.Unmarshal(data, config) != nil {
		return false
	}
	switch channel {
	case notification.Discord:
		return config.GetDiscord().GetEncryptedWebhookUrl() != ""
	case notification.Slack:
		return config.GetSlack().GetEncryptedWebhookUrl() != ""
	case notification.Webhook:
		return config.GetWebhook().GetEncryptedUrl() != "" && config.GetWebhook().GetEncryptedSigningSecret() != ""
	case notification.Email:
		return false
	}
	return false
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
