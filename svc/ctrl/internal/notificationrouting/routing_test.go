package notificationrouting

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"google.golang.org/protobuf/proto"
)

func TestRecipientsDeduplicateAudienceAndDirectTargetsAndKeepSuppressionReasons(t *testing.T) {
	adminID, memberID, departedID := uid.New(uid.TestPrefix), uid.New(uid.TestPrefix), uid.New(uid.TestPrefix)
	destinationID := uid.New(uid.TestPrefix)
	members := []workos.Member{{UserID: adminID, Email: "admin@example.com", Admin: true}, {UserID: memberID, Email: "member@example.com"}}
	admins, err := WorkspaceAdmins(t.Context(), uid.New(uid.WorkspacePrefix), members)
	require.NoError(t, err)
	require.Equal(t, []string{adminID}, admins)
	config, err := proto.Marshal(&notificationv1.DestinationConfig{Destination: &notificationv1.DestinationConfig_Slack{Slack: &notificationv1.SlackConfig{EncryptedWebhookUrl: "ciphertext"}}})
	require.NoError(t, err)
	targets := []db.ListNotificationTargetsRow{
		{Channel: db.NotificationSubscriptionTargetsChannelEmail, Audience: sql.NullInt32{Int32: int32(notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS), Valid: true}},
		{Channel: db.NotificationSubscriptionTargetsChannelEmail, UserID: nullable(adminID)},
		{Channel: db.NotificationSubscriptionTargetsChannelEmail, UserID: nullable(memberID)},
		{Channel: db.NotificationSubscriptionTargetsChannelEmail, UserID: nullable(departedID)},
		{Channel: db.NotificationSubscriptionTargetsChannelSlack, DestinationID: nullable(destinationID), DestinationConfig: nullable(string(config)), DestinationEnabled: sql.NullBool{Bool: true, Valid: true}},
	}
	preferences := []db.ListNotificationPreferencesRow{{UserID: adminID, Enabled: false}}
	recipients, err := resolveTargets(targets, members, preferences, map[notificationv1.Audience][]string{notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS: admins})
	require.NoError(t, err)
	require.Len(t, recipients, 4)
	byKey := make(map[string]Recipient)
	for _, recipient := range recipients {
		byKey[recipient.Key] = recipient
	}
	require.Equal(t, "email_opted_out", byKey["user:"+adminID].Reason)
	require.Equal(t, "member_unavailable", byKey["user:"+departedID].Reason)
	require.Empty(t, byKey["user:"+memberID].Reason)
	require.Equal(t, notification.Slack, byKey["destination:"+destinationID].Channel)
	require.Equal(t, config, byKey["destination:"+destinationID].Config)
	require.Empty(t, byKey["destination:"+destinationID].Reason)
}

func TestDestinationSuppressionAndRecipientLimit(t *testing.T) {
	for _, test := range []struct {
		name    string
		enabled sql.NullBool
		reason  string
	}{
		{"unavailable", sql.NullBool{}, "destination_unavailable"},
		{"paused", sql.NullBool{Valid: true}, "destination_paused"},
		{"invalid config", sql.NullBool{Bool: true, Valid: true}, "invalid_destination_config"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recipients, err := resolveTargets([]db.ListNotificationTargetsRow{{Channel: db.NotificationSubscriptionTargetsChannelWebhook, DestinationID: nullable(uid.New(uid.TestPrefix)), DestinationEnabled: test.enabled}}, nil, nil, nil)
			require.NoError(t, err)
			require.Len(t, recipients, 1)
			require.Equal(t, test.reason, recipients[0].Reason)
		})
	}
	targets := make([]db.ListNotificationTargetsRow, 1001)
	for index := range targets {
		targets[index] = db.ListNotificationTargetsRow{Channel: db.NotificationSubscriptionTargetsChannelEmail, UserID: nullable(uid.New(uid.TestPrefix))}
	}
	_, err := resolveTargets(targets, nil, nil, nil)
	var invalid *InvalidError
	require.ErrorAs(t, err, &invalid)
	require.Equal(t, "notification recipient limit exceeded", invalid.Reason)
}
