package notification

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/proto"
)

type routingDatabase struct {
	db.Database
	targets []db.ListNotificationTargetsRow
}

func (d routingDatabase) ListNotificationTargets(context.Context, db.ListNotificationTargetsParams) ([]db.ListNotificationTargetsRow, error) {
	return d.targets, nil
}

func TestWorkerPreservesSharedDestinationResolution(t *testing.T) {
	id := uid.New(uid.TestPrefix)
	config, err := proto.Marshal(&notificationv1.DestinationConfig{Destination: &notificationv1.DestinationConfig_Discord{Discord: &notificationv1.DiscordConfig{EncryptedWebhookUrl: "ciphertext"}}})
	require.NoError(t, err)
	s := &Service{db: routingDatabase{targets: []db.ListNotificationTargetsRow{{
		Channel:            db.NotificationSubscriptionTargetsChannelDiscord,
		DestinationID:      nullable(id),
		DestinationEnabled: sql.NullBool{Bool: true, Valid: true},
		DestinationConfig:  nullable(string(config)),
	}}}}
	recipients, err := s.resolve(t.Context(), notification.Event{WorkspaceID: uid.New(uid.WorkspacePrefix)})
	require.NoError(t, err)
	require.Len(t, recipients, 1)
	require.Equal(t, recipient{key: "destination:" + id, channel: notification.Discord, config: config}, recipients[0])
}
