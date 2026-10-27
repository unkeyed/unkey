package notification

import (
	"context"
	"errors"

	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/proto"
)

func (s *Service) allowed(ctx context.Context, frozen snapshot, channel notification.Channel) (bool, error) {
	valid, err := s.db.NotificationResourceBelongsToWorkspace(ctx, db.NotificationResourceBelongsToWorkspaceParams{
		WorkspaceID:   frozen.Event.WorkspaceID,
		ProjectID:     frozen.Event.Resource.ProjectID,
		AppID:         frozen.Event.Resource.AppID,
		EnvironmentID: frozen.Event.Resource.EnvironmentID,
	})
	if err != nil || !valid {
		return false, err
	}
	if channel != notification.Email {
		return s.db.NotificationDestinationEnabled(ctx, db.NotificationDestinationEnabledParams{
			WorkspaceID: frozen.Event.WorkspaceID,
			ID:          frozen.DestinationID,
			Channel:     db.NotificationDestinationsChannel(channel),
		})
	}

	enabled, err := s.db.NotificationEmailEnabled(ctx, db.NotificationEmailEnabledParams{
		WorkspaceID:      frozen.Event.WorkspaceID,
		UserID:           frozen.UserID,
		NotificationType: frozen.Event.Type,
	})
	if err != nil || !enabled {
		return false, err
	}
	if s.directory == nil {
		return false, errors.New("notification member directory unavailable")
	}
	workspace, err := s.db.FindWorkspaceByID(ctx, frozen.Event.WorkspaceID)
	if err != nil {
		return false, err
	}
	return s.directory.CanEmail(ctx, workspace.OrgID, frozen.UserID, frozen.Email)
}

func (s *Service) target(ctx context.Context, workspaceID string, frozen snapshot, channel notification.Channel) (notification.Target, error) {
	target := notification.Target{Email: frozen.Email, URL: "", Headers: nil, SigningSecret: ""}
	if channel == notification.Email {
		return target, nil
	}
	config := &notificationv1.DestinationConfig{}
	if proto.Unmarshal(frozen.Config, config) != nil || !validDestination(channel, frozen.Config) {
		return target, invalid("invalid destination snapshot")
	}
	var encryptedURL string
	switch channel {
	case notification.Slack:
		encryptedURL = config.GetSlack().GetEncryptedWebhookUrl()
	case notification.Discord:
		encryptedURL = config.GetDiscord().GetEncryptedWebhookUrl()
	case notification.Webhook:
		webhook := config.GetWebhook()
		encryptedURL = webhook.GetEncryptedUrl()
		var err error
		target.SigningSecret, err = s.decrypt(ctx, workspaceID, webhook.GetEncryptedSigningSecret())
		if err != nil {
			return target, err
		}
		target.Headers = make(map[string]string, len(webhook.GetHeaders()))
		for _, header := range webhook.GetHeaders() {
			value, err := s.decrypt(ctx, workspaceID, header.GetEncryptedValue())
			if err != nil {
				return target, err
			}
			target.Headers[header.GetName()] = value
		}
	case notification.Email:
		return target, invalid("email is not a destination")
	}
	var err error
	target.URL, err = s.decrypt(ctx, workspaceID, encryptedURL)
	return target, err
}

func (s *Service) decrypt(ctx context.Context, workspaceID, ciphertext string) (string, error) {
	response, err := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{Keyring: workspaceID, Encrypted: ciphertext})
	if err != nil {
		return "", errors.New("decrypt notification target")
	}
	return response.GetPlaintext(), nil
}
