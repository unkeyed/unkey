package notification

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/notificationrouting"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

type recipient struct {
	key     string
	channel notification.Channel
	email   string
	config  []byte
	reason  string
}

func WorkspaceAdmins(ctx context.Context, workspaceID string, members []workos.Member) ([]string, error) {
	return notificationrouting.WorkspaceAdmins(ctx, workspaceID, members)
}

func (s *Service) resolve(ctx context.Context, event notification.Event) ([]recipient, error) {
	resolver := notificationrouting.Resolver{Database: s.db, Directory: s.directory, Audiences: s.audiences}
	resolved, err := resolver.Resolve(ctx, event)
	if err != nil {
		var invalidRouting *notificationrouting.InvalidError
		if errors.As(err, &invalidRouting) {
			return nil, invalid(invalidRouting.Reason)
		}
		return nil, err
	}
	recipients := make([]recipient, 0, len(resolved))
	for _, target := range resolved {
		recipients = append(recipients, recipient{key: target.Key, channel: target.Channel, email: target.Email, config: target.Config, reason: target.Reason})
	}
	return recipients, nil
}

func validDestination(channel notification.Channel, data []byte) bool {
	return notificationrouting.ValidDestination(channel, data)
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
